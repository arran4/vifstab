package diagram

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/disk"
)

// imageMetadata is a metadata-only handle. ReadOnly may be called only after
// Stat proves that this exact object is a regular file, never a device or FIFO.
type imageMetadata interface {
	Stat() (fs.FileInfo, error)
	ReadOnly() (backend.File, error)
	Close() error
}

func regularImageFile(path string, openMetadata func(string) (imageMetadata, error)) (backend.File, error) {
	metadata, err := openMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("image metadata unknown: %w", err)
	}
	if nilPointer(metadata) {
		return nil, fmt.Errorf("image metadata unknown: no metadata handle")
	}
	info, err := metadata.Stat()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("image metadata unknown: %w", err), metadata.Close())
	}
	if nilPointer(info) || !info.Mode().IsRegular() {
		return nil, errors.Join(fmt.Errorf("image input is not a regular file; no data handle opened"), metadata.Close())
	}
	reader, err := metadata.ReadOnly()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("regular-image read access unavailable: %w", err), metadata.Close())
	}
	if nilPointer(reader) {
		return nil, errors.Join(fmt.Errorf("regular-image read access unavailable: no data handle"), metadata.Close())
	}
	opened, err := reader.Stat()
	if err != nil || nilPointer(opened) || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.Join(fmt.Errorf("image identity could not be verified after opening"), err, reader.Close(), metadata.Close())
	}
	if err := metadata.Close(); err != nil {
		return nil, errors.Join(fmt.Errorf("close image metadata handle: %w", err), reader.Close())
	}
	return reader, nil
}

func openRegularImage(path string, opts ...diskfs.OpenOpt) (*disk.Disk, error) {
	reader, err := regularImageFile(path, openImageMetadata)
	if err != nil {
		return nil, err
	}
	// File.New has no source path and refuses Writable. The parser can only
	// access the already verified regular file, never reopen the input pathname.
	d, err := diskfs.OpenBackend(file.New(reader, true), opts...)
	if err != nil {
		return nil, errors.Join(err, reader.Close())
	}
	return d, nil
}
