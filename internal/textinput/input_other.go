//go:build !linux

package textinput

import "fmt"

func openMetadata(string) (metadata, error) {
	return nil, fmt.Errorf("safe text input opening requires Linux metadata/proc-fd interfaces")
}
