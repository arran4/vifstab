package rules

import (
	"context"
	"io/fs"
	"os"
)

type API interface {
	CheckFileExists(path string) bool
	CheckBlockDeviceExists(dev string) bool
	GetHostKernelVersion() string
}

type ActualFS struct{}

func (ActualFS) CheckFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (ActualFS) CheckBlockDeviceExists(dev string) bool {
	resolvedPath := ResolveDevicePath(dev)
	_, err := os.Stat(resolvedPath)
	return err == nil
}



func (ActualFS) GetHostKernelVersion() string {
	b, err := os.ReadFile("/proc/version")
	if err == nil {
		return string(b)
	}
	b, err = os.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil {
		return string(b)
	}
	return ""
}

type CacheFS struct {
	Cache *Cache
	FS    fs.FS
}

func (c *CacheFS) GetHostKernelVersion() string {
	if c.Cache != nil {
		if v, ok := c.Cache.Get("proc_version"); ok {
			if b, ok := v.([]byte); ok {
				return string(b)
			} else if s, ok := v.(string); ok {
				return s
			}
		}
		if v, ok := c.Cache.Get("proc_osrelease"); ok {
			if b, ok := v.([]byte); ok {
				return string(b)
			} else if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ActualFS{}.GetHostKernelVersion()
}

func (c *CacheFS) CheckFileExists(path string) bool {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + path)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	// Fallback to real OS using the raw path
	return ActualFS{}.CheckFileExists(path)
}

func (c *CacheFS) CheckBlockDeviceExists(dev string) bool {
	resolvedPath := ResolveDevicePath(dev)

	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + resolvedPath)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
		// Also check if they cached the raw path for some reason
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + dev)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	// Fallback to real OS using the resolved path
	return ActualFS{}.CheckBlockDeviceExists(dev)
}
