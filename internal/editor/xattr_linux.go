//go:build linux

package editor

import (
	"golang.org/x/sys/unix"
)

func osListxattr(path string) ([]string, error) {
	sz, err := unix.Listxattr(path, nil)
	if err != nil || sz <= 0 {
		return nil, err
	}
	buf := make([]byte, sz)
	n, err := unix.Listxattr(path, buf)
	if err != nil {
		return nil, err
	}
	buf = buf[:n]
	var attrs []string
	start := 0
	for i, b := range buf {
		if b == 0 {
			if i > start {
				attrs = append(attrs, string(buf[start:i]))
			}
			start = i + 1
		}
	}
	return attrs, nil
}

func osGetxattr(path, name string) ([]byte, error) {
	sz, err := unix.Getxattr(path, name, nil)
	if err != nil {
		return nil, err
	}
	if sz == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, sz)
	n, err := unix.Getxattr(path, name, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func osSetxattr(path, name string, value []byte) error {
	return unix.Setxattr(path, name, value, 0)
}
