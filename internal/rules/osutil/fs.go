package osutil

import (
	"bufio"
	"os"
	"strings"
)

func LoadProcFilesystems(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fsMap := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 {
			fsMap[fields[len(fields)-1]] = true
		}
	}
	return fsMap, scanner.Err()
}
