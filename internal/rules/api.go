package rules

import (
	"context"
	"io/fs"
	"os"
)

type API interface {
	CheckFileExists(path string) bool
}

type ActualFS struct{}

func (ActualFS) CheckFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type CacheFS struct {
	Cache *Cache
	FS    fs.FS
}

func (c CacheFS) CheckFileExists(path string) bool {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + path)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	// Fallback to real OS
	return ActualFS{}.CheckFileExists(path)
}

func APIFromContext(ctx context.Context) API {
	c := CacheFromContext(ctx)
	if c != nil {
		return CacheFS{Cache: c}
	}
	return ActualFS{}
}
