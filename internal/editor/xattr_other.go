//go:build !linux

package editor

import "errors"

func osListxattr(path string) ([]string, error) {
	return nil, errors.New("xattrs unsupported on non-Linux")
}

func osGetxattr(path, name string) ([]byte, error) {
	return nil, errors.New("xattrs unsupported on non-Linux")
}

func osSetxattr(path, name string, value []byte) error {
	return errors.New("xattrs unsupported on non-Linux")
}

func osRemovexattr(path, name string) error {
	return errors.New("xattrs unsupported on non-Linux")
}
