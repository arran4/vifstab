package rules

import (
	"io/fs"
	"os"
)

import (
	"bufio"
	"strconv"
	"strings"
)

type FileExistsResult int

const (
	FileExists FileExistsResult = iota
	FileDoesNotExist
	FilePermissionDenied
)

type API interface {
	CheckFileExists(path string) FileExistsResult
	CheckBlockDeviceExists(dev string) bool
	GetHostKernelVersion() string
	CheckUserExists(uid int) bool
	CheckGroupExists(gid int) bool
}

type ActualFS struct{}

func (ActualFS) CheckUserExists(uid int) bool {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return false
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			if id, err := strconv.Atoi(parts[2]); err == nil && id == uid {
				return true
			}
		}
	}
	return false
}

func (ActualFS) CheckGroupExists(gid int) bool {
	f, err := os.Open("/etc/group")
	if err != nil {
		return false
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			if id, err := strconv.Atoi(parts[2]); err == nil && id == gid {
				return true
			}
		}
	}
	return false
}

func (ActualFS) CheckFileExists(path string) FileExistsResult {
	_, err := os.Stat(path)
	if err == nil {
		return FileExists
	}
	if os.IsPermission(err) {
		return FilePermissionDenied
	}
	return FileDoesNotExist
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

func NewCacheFS(cache *Cache, mockFS fs.FS) *CacheFS {
	if mockFS == nil {
		panic("CacheFS must be initialized with a valid fs.FS")
	}
	return &CacheFS{
		Cache: cache,
		FS:    mockFS,
	}
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

func (c *CacheFS) CheckFileExists(path string) FileExistsResult {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + path)); ok {
			if exists, ok := v.(bool); ok {
				if exists {
					return FileExists
				}
				return FileDoesNotExist
			}
			if result, ok := v.(FileExistsResult); ok {
				return result
			}
		}
	}
	if c.FS != nil {
		if path != "" && path[0] == '/' {
			path = path[1:]
		}
		_, err := fs.Stat(c.FS, path)
		if err == nil {
			return FileExists
		}
		if os.IsPermission(err) {
			return FilePermissionDenied
		}
		return FileDoesNotExist
	}
	panic("CacheFS must be initialized with a valid fs.FS")
}

func (c *CacheFS) CheckUserExists(uid int) bool {
	if c.Cache != nil {
		cacheKey := CacheKey("user_exists:" + strconv.Itoa(uid))
		if v, ok := c.Cache.Get(cacheKey); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	if c.FS != nil {
		f, err := c.FS.Open("etc/passwd")
		if err != nil {
			return false
		}
		defer func() {
			_ = f.Close()
		}()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ":")
			if len(parts) >= 3 {
				if id, err := strconv.Atoi(parts[2]); err == nil && id == uid {
					if c.Cache != nil {
						c.Cache.Set(CacheKey("user_exists:"+strconv.Itoa(uid)), true)
					}
					return true
				}
			}
		}
		if c.Cache != nil {
			c.Cache.Set(CacheKey("user_exists:"+strconv.Itoa(uid)), false)
		}
	}
	return false
}

func (c *CacheFS) CheckGroupExists(gid int) bool {
	if c.Cache != nil {
		cacheKey := CacheKey("group_exists:" + strconv.Itoa(gid))
		if v, ok := c.Cache.Get(cacheKey); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	if c.FS != nil {
		f, err := c.FS.Open("etc/group")
		if err != nil {
			return false
		}
		defer func() {
			_ = f.Close()
		}()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ":")
			if len(parts) >= 3 {
				if id, err := strconv.Atoi(parts[2]); err == nil && id == gid {
					if c.Cache != nil {
						c.Cache.Set(CacheKey("group_exists:"+strconv.Itoa(gid)), true)
					}
					return true
				}
			}
		}
		if c.Cache != nil {
			c.Cache.Set(CacheKey("group_exists:"+strconv.Itoa(gid)), false)
		}
	}
	return false
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
	if c.FS != nil {
		checkPath := resolvedPath
		if checkPath != "" && checkPath[0] == '/' {
			checkPath = checkPath[1:]
		}
		_, err := fs.Stat(c.FS, checkPath)
		return err == nil
	}
	panic("CacheFS must be initialized with a valid fs.FS")
}
