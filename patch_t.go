package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	d, err := os.MkdirTemp("", "vifstab-*")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Dir:", d)
	fmt.Println("HasPrefix:", strings.HasPrefix(d, os.TempDir()))
	os.RemoveAll(d)
}
