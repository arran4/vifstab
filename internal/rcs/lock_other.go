//go:build !linux

package rcs

import (
	"fmt"
	"io"
)

func LockOSFile(string) (io.Closer, error) {
	return nil, fmt.Errorf("RCS history locking is currently supported on Linux only")
}

func ReadOSHistory(string) ([]byte, error) {
	return nil, fmt.Errorf("RCS recovery is currently supported on Linux only")
}
