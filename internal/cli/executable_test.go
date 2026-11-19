package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

// These scenarios exercise the generated binary. Its build-time default points
// at a fixture in its working directory; bare invocation never reads host fstab.
func TestExecutableTXTAR(t *testing.T) {
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "vifstab")
	build := exec.Command("go", "build", "-ldflags=-X github.com/arran4/vifstab.DefaultFile=fstab", "-o", binary, "../../cmd/vifstab")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	probe := filepath.Join(binDir, "editor-probe")
	build = exec.Command("go", "build", "-o", probe, "./testdata/editor")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build probe: %v\n%s", err, out)
	}
	for _, name := range []string{"systemctl", "sudo", "su", "doas", "pkexec", "run0", "mount", "umount", "lsblk", "blkid", "udevadm", "fdisk", "efibootmgr", "bootctl", "sh", "bash"} {
		if err := os.Symlink(probe, filepath.Join(binDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	fixtures, err := filepath.Glob("testdata/commands/*.txtar")
	if err != nil || len(fixtures) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			archive, err := txtar.ParseFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			sections := map[string][]byte{}
			for _, file := range archive.Files {
				if _, exists := sections[file.Name]; exists {
					t.Fatalf("duplicate section %s", file.Name)
				}
				sections[file.Name] = file.Data
			}
			var args []string
			if err := json.Unmarshal(sections["args.json"], &args); err != nil {
				t.Fatal(err)
			}
			// JSON string sections embed nested TXTAR without introducing
			// section delimiters into the outer executable fixture.
			for _, name := range []string{"stdin", "stdout", "stderr"} {
				if data, ok := sections[name+".json"]; ok {
					var value string
					if err := json.Unmarshal(data, &value); err != nil {
						t.Fatal(err)
					}
					sections[name] = []byte(value)
				}
			}
			dir := t.TempDir()
			fstab := filepath.Join(dir, "fstab")
			if err := os.WriteFile(fstab, sections["fstab"], 0640); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(fstab)
			if err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(dir, "launched.json")
			command := exec.Command(binary, args...)
			command.Dir = dir
			command.Stdin = bytes.NewReader(sections["stdin"])
			command.Env = []string{"PATH=" + binDir, "HOME=", "XDG_CACHE_HOME=", "USER=fixture", "VISUAL=" + probe, "EDITOR=" + probe, "TMPDIR=" + dir, "VIFSTAB_PROBE=" + marker, "VIFSTAB_EDITOR_EXIT=" + strings.TrimSpace(string(sections["editor-exit"]))}
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			status := 0
			if err := command.Run(); err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				status = exit.ExitCode()
			}
			wantStatus, err := strconv.Atoi(strings.TrimSpace(string(sections["status"])))
			if err != nil {
				t.Fatal(err)
			}
			if status != wantStatus || stdout.String() != string(sections["stdout"]) || stderr.String() != string(sections["stderr"]) {
				t.Fatalf("status %d want %d\nstdout: %q\nwant: %q\nstderr: %q\nwant: %q", status, wantStatus, stdout.String(), sections["stdout"], stderr.String(), sections["stderr"])
			}
			if string(sections["editor"]) == "yes\n" {
				data, err := os.ReadFile(marker)
				if err != nil {
					t.Fatalf("editor did not launch: %v", err)
				}
				var child struct {
					UID, GID int
					Args     []string
				}
				if err := json.Unmarshal(data, &child); err != nil {
					t.Fatal(err)
				}
				if child.UID != os.Geteuid() || child.GID != os.Getegid() || len(child.Args) != 2 || child.Args[0] != "--" || !strings.HasPrefix(child.Args[1], dir+"/vifstab-") {
					t.Fatalf("unexpected editor credentials or arguments: %+v", child)
				}
			} else {
				if _, err := os.Stat(marker); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("unexpected subprocess: %v", err)
				}
				after, err := os.Stat(fstab)
				if err != nil {
					t.Fatal(err)
				}
				content, err := os.ReadFile(fstab)
				if err != nil {
					t.Fatal(err)
				}
				if !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) || !bytes.Equal(content, sections["fstab"]) {
					t.Fatal("read-only route modified fstab")
				}
				files, err := os.ReadDir(dir)
				wantFiles := 1
				if data, ok := sections["capture.json"]; ok {
					wantFiles = 2
					var expected string
					if err := json.Unmarshal(data, &expected); err != nil {
						t.Fatal(err)
					}
					archive, err := os.ReadFile(filepath.Join(dir, "case.txtar"))
					if err != nil || string(archive) != expected {
						t.Fatalf("capture artifact mismatch: %v\ngot %q\nwant %q", err, archive, expected)
					}
					info, err := os.Stat(filepath.Join(dir, "case.txtar"))
					if err != nil || info.Mode().Perm() != 0600 {
						t.Fatalf("capture permissions: %v, %v", info, err)
					}
				}
				if err != nil || len(files) != wantFiles {
					t.Fatalf("unexpected files: %v, %v", files, err)
				}
				for _, file := range files {
					if file.Name() != "fstab" && (wantFiles != 2 || file.Name() != "case.txtar") {
						t.Fatalf("unexpected file %s", file.Name())
					}
				}
			}
		})
	}
}
