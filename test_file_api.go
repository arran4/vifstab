package main

import (
	"fmt"
	"os"
)

func main() {
    f, err := os.Open("/etc/shadow")
    if err != nil {
        if os.IsPermission(err) {
            fmt.Println("Permission denied")
        } else if os.IsNotExist(err) {
            fmt.Println("Does not exist")
        }
    } else {
        fmt.Println("Exists")
        f.Close()
    }
}
