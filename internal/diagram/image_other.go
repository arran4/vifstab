//go:build !linux

package diagram

import "fmt"

func openImageMetadata(string) (imageMetadata, error) {
	return nil, fmt.Errorf("safe regular-image opening is unavailable on this platform")
}
