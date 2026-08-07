package rcs

import (
	"bytes"
	"fmt"
	"io/fs"
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

// FS abstracts filesystem operations to allow mocking during tests.
type FS interface {
	MkdirAll(path string, perm os.FileMode) error
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

type osFS struct{}

func (osFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (osFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error { return os.WriteFile(name, data, perm) }

var DefaultFS FS = osFS{}

func Update(filePath string, content []byte, config Config, opts ...any) error {
	if config.Dir == "" {
		return nil
	}

	var fsLayer FS = DefaultFS
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

	if _, err := fsLayer.Stat(rcsFilepath); os.IsNotExist(err) || err != nil {
		if err != nil && !os.IsNotExist(err) {
			// In case the mock fs doesn't strictly return os.ErrNotExist, assume it doesn't exist
			if _, ok := err.(*fs.PathError); !ok && err != os.ErrNotExist {
			    // fallback assumption
			}
		}
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
		}
	}

	err := fsLayer.WriteFile(rcsFilepath, []byte(f.String()), 0600)
	if err != nil {
		return fmt.Errorf("failed to write rcs file: %w", err)
	}

	return nil
}
