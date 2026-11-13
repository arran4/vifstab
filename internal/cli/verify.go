package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/textinput"
	"github.com/arran4/vifstab/internal/txtar"
)

func verify(file, snapshot, capture string, static, unredacted bool, seen map[string]bool) error {
	if seen["snapshot"] && snapshot == "" {
		return fmt.Errorf("--snapshot requires a non-empty input path")
	}
	if seen["capture"] && capture == "" {
		return fmt.Errorf("--capture requires a non-empty output path")
	}
	if snapshot != "" {
		if capture != "" || seen["static"] || seen["file"] || seen["unredacted"] {
			return fmt.Errorf("--snapshot cannot be combined with --file, --capture, --static or --unredacted")
		}
		return verifySnapshot(snapshot)
	}
	if unredacted && capture == "" {
		return fmt.Errorf("--unredacted requires --capture")
	}
	var api rules.API = rules.ActualFS{}
	if static {
		api = rules.StaticAPI{}
	}
	var report io.Writer = os.Stdout
	if capture == "-" {
		report = os.Stderr
	}
	if static {
		if _, err := fmt.Fprintln(report, "Static verification: live observations are unchecked/unknown."); err != nil {
			return err
		}
	}
	input, closeInput, err := openInput(file)
	if err != nil {
		return err
	}
	defer closeInput()
	if capture != "" {
		if unredacted {
			fmt.Fprintln(os.Stderr, "Warning: unredacted capture includes sensitive identifiers, paths and credentials. Review before sharing.")
		} else {
			fmt.Fprintln(os.Stderr, "Capture redacts private values; review before sharing. Redaction can change validation findings.")
		}
		result, err := txtar.CaptureReader(context.Background(), api, input, unredacted)
		if err != nil {
			return fmt.Errorf("capture failed: %w", err)
		}
		if err := writeCapture(capture, result.Archive); err != nil {
			return err
		}
		if result.ValidationError != nil {
			return fmt.Errorf("capture saved; verification failed: %w", result.ValidationError)
		}
		return Report(report, result.Issues)
	}
	issues, err := editor.VerifyAndLintReader(input, api)
	if err != nil {
		return applicationError(err)
	}
	return Report(report, issues)
}

// openInput borrows stdin only when explicitly selected. File ownership remains
// with this helper, not with the generated I/O bindings.
func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		if err := textinput.Check(os.Stdin); err != nil {
			return nil, func() {}, fmt.Errorf("open stdin: %w", err)
		}
		return os.Stdin, func() {}, nil
	}
	f, err := textinput.Open(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open input: %w", err)
	}
	return f, func() { _ = f.Close() }, nil
}

func verifySnapshot(path string) error {
	input, closeInput, err := openInput(path)
	if err != nil {
		return fmt.Errorf("failed to open txtar file: %w", err)
	}
	defer closeInput()
	data, err := txtar.Read(input)
	if err != nil {
		return fmt.Errorf("failed to read txtar input: %w", err)
	}
	snapshot, err := txtar.ParseTXTAR(data)
	if err != nil {
		return fmt.Errorf("failed to parse txtar file: %w", err)
	}
	api, err := txtar.ReplaySnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("failed to setup replay environment: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, "Snapshot verification: missing observations remain unknown; only recorded host data is used."); err != nil {
		return err
	}
	issues, err := editor.VerifyAndLintReader(bytes.NewReader(snapshot.Fstab), api)
	if err != nil {
		return err
	}
	return Report(os.Stdout, issues)
}

// writeCapture deliberately bypasses ordinary generated io.Writer opening:
// existing files and symlinks are rejected and private permissions are retained.
func writeCapture(path string, data []byte) error {
	if path == "-" {
		if _, err := os.Stdout.Write(data); err != nil {
			return fmt.Errorf("write capture output to stdout: %w", err)
		}
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("open capture output: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write capture output: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close capture output: %w", err)
	}
	return nil
}
