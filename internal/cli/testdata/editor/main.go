// This executable records inherited credentials and detects unexpected launches.
package main

import (
	"encoding/json"
	"os"
	"strconv"
)

func main() {
	data, err := json.Marshal(struct {
		UID, GID int
		Args     []string
	}{os.Geteuid(), os.Getegid(), os.Args[1:]})
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Getenv("VIFSTAB_PROBE"), data, 0600); err != nil {
		panic(err)
	}
	code, _ := strconv.Atoi(os.Getenv("VIFSTAB_EDITOR_EXIT"))
	os.Exit(code)
}
