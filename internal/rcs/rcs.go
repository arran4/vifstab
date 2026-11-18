package rcs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-rcs"
)

type Config struct {
	Dir string
}

type TempFile interface {
	io.WriteCloser
	Name() string
	Chmod(mode os.FileMode) error
	Sync() error
}

// FS abstracts filesystem operations to allow mocking during tests.
type FS interface {
	MkdirAll(path string, perm os.FileMode) error
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	Getwd() (string, error)
	Lock(name string) (io.Closer, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	CreateTemp(dir, pattern string) (TempFile, error)
	Rename(oldpath, newpath string) error
	Remove(name string) error
	SyncDir(name string) error
}

type osFS struct{}

func (osFS) MkdirAll(path string, perm os.FileMode) error { return MakeDurableDirs(path, perm) }
func (osFS) Stat(name string) (os.FileInfo, error)        { return os.Stat(name) }
func (osFS) Lstat(name string) (os.FileInfo, error)       { return os.Lstat(name) }
func (osFS) Getwd() (string, error)                       { return os.Getwd() }
func (osFS) Lock(name string) (io.Closer, error)          { return LockOSFile(name) }
func (osFS) ReadFile(name string) ([]byte, error)         { return ReadOSHistory(name) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFS) CreateTemp(dir, pattern string) (TempFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (osFS) Rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
func (osFS) Remove(name string) error             { return os.Remove(name) }
func (osFS) SyncDir(name string) error {
	d, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}

var DefaultFS FS = osFS{}

func Update(filePath string, content []byte, config Config, opts ...any) (resultErr error) {
	if config.Dir == "" {
		return nil
	}

	var fsLayer = DefaultFS
	for _, opt := range opts {
		if injection, ok := opt.(FS); ok {
			fsLayer = injection
		}
	}

	identity, rcsFilepath, err := historyIdentity(filePath, config.Dir, fsLayer)
	if err != nil {
		return err
	}
	if err := fsLayer.MkdirAll(historyParent(rcsFilepath), 0700); err != nil {
		return fmt.Errorf("failed to create rcs dir: %w", err)
	}
	lock, err := fsLayer.Lock(rcsFilepath + ".lock")
	if err != nil {
		return fmt.Errorf("failed to lock rcs history: %w", err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("failed to release rcs lock: %w", err))
		}
	}()

	var f *rcs.File

	info, err := fsLayer.Lstat(rcsFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			legacy := joinHistoryPath(config.Dir, filepath.Base(filePath)+",v")
			if _, err := fsLayer.Lstat(legacy); err == nil {
				return &LegacyHistoryError{LegacyPath: legacy, HistoryPath: rcsFilepath}
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to inspect legacy rcs history: %w", err)
			}
			// Treat as new file
			f = rcs.NewFile()
			f.Head = "1.1"
			f.Access = true
			f.Strict = true
			if err := setProvenance(f, identity, "1.1", content, nil); err != nil {
				return err
			}

			f.RevisionHeads = append(f.RevisionHeads, &rcs.RevisionHead{
				Revision: "1.1",
				Date:     rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
				Author:   archiveAuthor(),
				State:    "Exp",
			})

			f.RevisionContents = append(f.RevisionContents, &rcs.RevisionContent{
				Revision: "1.1",
				Log:      "Initial revision",
				Text:     string(content),
			})
		} else {
			return fmt.Errorf("failed to stat rcs file: %w", err)
		}
	} else {
		if info == nil || !info.Mode().IsRegular() {
			return fmt.Errorf("RCS history must be a regular file")
		}
		rcsData, err := fsLayer.ReadFile(rcsFilepath)
		if err != nil {
			return fmt.Errorf("failed to open rcs file: %w", err)
		}

		f, err = rcs.ParseFile(bytes.NewReader(rcsData))
		if err != nil {
			return fmt.Errorf("failed to parse rcs file: %w", err)
		}
		provenance, err := parseProvenance(f, identity)
		if err != nil {
			return err
		}
		previous, err := restoreRevision(f, provenance, f.Head)
		if err != nil {
			return fmt.Errorf("failed to verify previous rcs head: %w", err)
		}

		headParts := strings.Split(f.Head, ".")
		if len(headParts) == 2 && headParts[0] == "1" {
			minor := headParts[1]
			newMinor, err := strconv.Atoi(minor)
			if err != nil {
				return fmt.Errorf("failed to parse minor revision %q: %w", minor, err)
			}
			if newMinor < 1 || newMinor == math.MaxInt {
				return fmt.Errorf("RCS revision counter is out of range")
			}
			newHead := fmt.Sprintf("%s.%d", headParts[0], newMinor+1)
			oldHead := f.Head
			if err := linkPreviousRevision(f, content, previous); err != nil {
				return err
			}
			f.Head = newHead
			if err := setProvenance(f, identity, newHead, content, provenance); err != nil {
				return err
			}

			f.RevisionHeads = append([]*rcs.RevisionHead{{
				Revision:     rcs.Num(newHead),
				Date:         rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
				Author:       archiveAuthor(),
				State:        "Exp",
				NextRevision: rcs.Num(oldHead),
			}}, f.RevisionHeads...)

			f.RevisionContents = append([]*rcs.RevisionContent{{
				Revision: newHead,
				Log:      "Updated revision",
				Text:     string(content),
			}}, f.RevisionContents...)
		} else {
			return fmt.Errorf("unsupported revision format: %s", f.Head)
		}
	}

	data := []byte(f.String())
	// Verify the serialization and complete recovery chain before replacing any
	// archive. A successful parse alone does not prove older bytes are recoverable.
	encoded, err := rcs.ParseFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to verify encoded RCS history: %w", err)
	}
	provenance, err := parseProvenance(encoded, identity)
	if err != nil {
		return err
	}
	restored, err := restoreRevision(encoded, provenance, encoded.Head)
	if err != nil {
		return fmt.Errorf("failed to verify encoded RCS recovery: %w", err)
	}
	if !bytes.Equal(restored, content) {
		return fmt.Errorf("encoded RCS head did not preserve content")
	}
	dir := historyParent(rcsFilepath)
	tmpDest, writeErr := fsLayer.CreateTemp(dir, "rcs-tmp-*")
	if writeErr != nil {
		return fmt.Errorf("failed to create temporary rcs file: %w", writeErr)
	}
	defer func() { _ = fsLayer.Remove(tmpDest.Name()) }()

	if n, err := tmpDest.Write(data); err != nil || n != len(data) {
		_ = tmpDest.Close()
		if err == nil {
			err = io.ErrShortWrite
		}
		return fmt.Errorf("failed to write to temporary rcs file: %w", err)
	}

	if err := tmpDest.Chmod(0600); err != nil {
		_ = tmpDest.Close()
		return fmt.Errorf("failed to chmod temporary rcs file: %w", err)
	}

	if err := tmpDest.Sync(); err != nil {
		_ = tmpDest.Close()
		return fmt.Errorf("failed to sync temporary rcs file: %w", err)
	}

	if err := tmpDest.Close(); err != nil {
		return fmt.Errorf("failed to close temporary rcs file: %w", err)
	}

	if err := fsLayer.Rename(tmpDest.Name(), rcsFilepath); err != nil {
		return fmt.Errorf("failed to replace rcs file: %w", err)
	}

	if err := fsLayer.SyncDir(dir); err != nil {
		return fmt.Errorf("failed to sync directory after replacing rcs file: %w", err)
	}

	return nil
}
