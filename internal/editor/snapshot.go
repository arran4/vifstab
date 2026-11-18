package editor

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

// ErrConflict means the destination differs from the pre-editor snapshot.
// No replacement or RCS update is performed for a detected conflict.
var ErrConflict = errors.New("destination changed since editing began; reconcile and retry")

type destinationSnapshot struct {
	target effectiveTarget
	digest [sha256.Size]byte
}

// snapshotDestination reads only through the injected filesystem. Checking the
// resolved path and stat identity around the read refuses an inconsistent view.
// This is optimistic detection; a non-cooperating writer can still race the
// final comparison and rename, which are not a filesystem transaction.
func snapshotDestination(name string, fs FS) (destinationSnapshot, []byte, error) {
	target, err := resolveTarget(name, fs)
	if err != nil {
		return destinationSnapshot{}, nil, err
	}
	var content []byte
	if target.info != nil {
		content, err = fs.ReadFile(target.path)
		if err != nil {
			return destinationSnapshot{}, nil, fmt.Errorf("failed to read destination snapshot: %w", err)
		}
	}
	after, err := resolveTarget(name, fs)
	if err != nil {
		return destinationSnapshot{}, nil, fmt.Errorf("failed to verify destination snapshot: %w", err)
	}
	if !sameTarget(target, after, fs) {
		return destinationSnapshot{}, nil, ErrConflict
	}
	return destinationSnapshot{target: target, digest: sha256.Sum256(content)}, content, nil
}

func sameTarget(a, b effectiveTarget, fs FS) bool {
	if a.path != b.path {
		return false
	}
	if a.info == nil || b.info == nil {
		return a.info == nil && b.info == nil
	}
	return fs.SameFile(a.info, b.info) && a.info.Size() == b.info.Size() &&
		a.info.ModTime().Equal(b.info.ModTime()) && a.info.Mode() == b.info.Mode()
}

func (s destinationSnapshot) check(name string, fs FS) error {
	current, _, err := snapshotDestination(name, fs)
	if err != nil {
		return fmt.Errorf("cannot verify destination before saving: %w", err)
	}
	if !sameTarget(s.target, current.target, fs) || s.digest != current.digest {
		return ErrConflict
	}
	return nil
}

// RecoveryError identifies a private copy of edited content after a refused
// save. Unwrap preserves the conflict or underlying operational error.
type RecoveryError struct {
	Path string
	Err  error
}

func (e *RecoveryError) Error() string {
	return fmt.Sprintf("%v; edited content preserved at %q", e.Err, e.Path)
}

func (e *RecoveryError) Unwrap() error { return e.Err }

func preserveRecovery(content []byte, fs FS) (path string, err error) {
	f, err := fs.CreateTemp("", "vifstab-recovery-*")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = fs.Remove(f.Name())
		}
	}()
	var n int
	if n, err = f.Write(content); err == nil && n != len(content) {
		err = io.ErrShortWrite
	}
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Chmod(0600); err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

var _ error = (*RecoveryError)(nil)
