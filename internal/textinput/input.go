// Package textinput opens configuration text without opening block devices for
// data. Callers retain the inherited execution context and own returned files.
package textinput

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

var ErrBlockDevice = errors.New("block devices are not text inputs")

// metadata is a stable metadata-only reference. Opening it must not invoke a
// block-device data-open operation. Data may be opened only after classification.
type metadata interface {
	Stat() (fs.FileInfo, error)
	ReadOnly() (fs.File, error)
	Close() error
}

// Check classifies an already supplied input without reading it. It permits
// regular files, named/anonymous pipes and character streams, including a tty.
// Directories, sockets, block devices and unavailable classification are errors.
// Reader-only APIs intentionally remain available for caller-supplied test data.
func Check(input interface{ Stat() (fs.FileInfo, error) }) error {
	if input == nil {
		return fmt.Errorf("text input type unknown: no input handle")
	}
	info, err := input.Stat()
	if err != nil {
		return fmt.Errorf("text input type unknown: %w", err)
	}
	return checkInfo(info)
}

func checkInfo(info fs.FileInfo) error {
	if info == nil {
		return fmt.Errorf("text input type unknown: no file information")
	}
	mode := info.Mode()
	if mode&fs.ModeDevice != 0 && mode&fs.ModeCharDevice == 0 {
		return ErrBlockDevice
	}
	switch mode.Type() {
	case 0, fs.ModeNamedPipe, fs.ModeDevice | fs.ModeCharDevice:
		return nil
	}
	return fmt.Errorf("unsupported text input type %s", mode.Type())
}

// Open reads through a stable object reference, never through a pathname that
// could be retargeted after checking. Symlinks retain their normal meaning.
func Open(path string) (fs.File, error) { return open(path, openMetadata) }

func open(path string, acquire func(string) (metadata, error)) (fs.File, error) {
	m, err := acquire(path)
	if err != nil {
		if m != nil {
			err = errors.Join(err, m.Close())
		}
		return nil, fmt.Errorf("text input metadata unavailable: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("text input metadata unavailable: no metadata handle")
	}
	info, err := m.Stat()
	if err == nil {
		err = checkInfo(info)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("classify text input: %w", err), m.Close())
	}
	f, err := m.ReadOnly()
	if err != nil {
		if f != nil {
			err = errors.Join(err, f.Close())
		}
		return nil, errors.Join(fmt.Errorf("text input read access unavailable: %w", err), m.Close())
	}
	if f == nil {
		return nil, errors.Join(fmt.Errorf("text input read access unavailable: no data handle"), m.Close())
	}
	opened, err := f.Stat()
	if err == nil {
		err = checkInfo(opened)
	}
	if err == nil && !os.SameFile(info, opened) {
		err = fmt.Errorf("text input identity changed")
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("verify opened text input: %w", err), f.Close(), m.Close())
	}
	if err := m.Close(); err != nil {
		return nil, errors.Join(fmt.Errorf("close text input metadata: %w", err), f.Close())
	}
	return f, nil
}

func ReadFile(path string) ([]byte, error) {
	f, err := Open(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(f)
	return data, errors.Join(readErr, f.Close())
}
