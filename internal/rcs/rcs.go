package rcs

import (
	"bytes"
	"fmt"
	"github.com/arran4/vifstab/internal/textinput"
	"io"
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
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	CreateTemp(dir, pattern string) (TempFile, error)
	Rename(oldpath, newpath string) error
	Remove(name string) error
	SyncDir(name string) error
}

type osFS struct{}

func (osFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osFS) Stat(name string) (os.FileInfo, error)        { return os.Stat(name) }
func (osFS) ReadFile(name string) ([]byte, error)         { return textinput.ReadFile(name) }
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

func Update(filePath string, content []byte, config Config, opts ...any) error {
	if config.Dir == "" {
		return nil
	}

	var fsLayer = DefaultFS
	for _, opt := range opts {
		if injection, ok := opt.(FS); ok {
			fsLayer = injection
		}
	}

	if err := fsLayer.MkdirAll(config.Dir, 0700); err != nil {
		return fmt.Errorf("failed to create rcs dir: %w", err)
	}

	baseName := filepath.Base(filePath)
	rcsFilepath := filepath.Join(config.Dir, baseName+",v")

	var f *rcs.File

	_, err := fsLayer.Stat(rcsFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Treat as new file
			f = rcs.NewFile()
			f.Head = "1.1"
			f.Access = true
			f.Strict = true

			f.RevisionHeads = append(f.RevisionHeads, &rcs.RevisionHead{
				Revision: "1.1",
				Date:     rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
				Author:   rcs.ID(os.Getenv("USER")),
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
		rcsData, err := fsLayer.ReadFile(rcsFilepath)
		if err != nil {
			return fmt.Errorf("failed to open rcs file: %w", err)
		}

		f, err = rcs.ParseFile(bytes.NewReader(rcsData))
		if err != nil {
			return fmt.Errorf("failed to parse rcs file: %w", err)
		}

		headParts := strings.Split(f.Head, ".")
		if len(headParts) == 2 {
			minor := headParts[1]
			newMinor, err := strconv.Atoi(minor)
			if err != nil {
				return fmt.Errorf("failed to parse minor revision %q: %w", minor, err)
			}
			newHead := fmt.Sprintf("%s.%d", headParts[0], newMinor+1)
			f.Head = newHead

			f.RevisionHeads = append([]*rcs.RevisionHead{{
				Revision: rcs.Num(newHead),
				Date:     rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
				Author:   rcs.ID(os.Getenv("USER")),
				State:    "Exp",
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

	dir := filepath.Dir(rcsFilepath)
	tmpDest, writeErr := fsLayer.CreateTemp(dir, "rcs-tmp-*")
	if writeErr != nil {
		return fmt.Errorf("failed to create temporary rcs file: %w", writeErr)
	}
	defer func() { _ = fsLayer.Remove(tmpDest.Name()) }()

	if _, err := tmpDest.Write([]byte(f.String())); err != nil {
		_ = tmpDest.Close()
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
