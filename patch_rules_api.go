package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	b, _ := ioutil.ReadFile("internal/rules/api.go")
	s := string(b)

	oldBlock := `		// Fallback check the original requested path on file_exists
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}

		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}`

	newBlock := `		// Fallback check the original requested path on file_exists
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return Exists
				}
				return DoesNotExist
			}
			if result, ok := v.(ExistState); ok {
				return result
			}
		}`

	if strings.Contains(s, oldBlock) {
		s = strings.Replace(s, oldBlock, newBlock, 1)
		ioutil.WriteFile("internal/rules/api.go", []byte(s), 0644)
	} else {
		// If not found, log it so we know
		panic("rules/api.go oldBlock not found")
	}
}
