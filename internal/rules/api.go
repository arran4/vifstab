package rules

import (
	"io/fs"
	"os"
)

import (
	"bufio"
	"path/filepath"
	"strconv"
	"strings"
)

type FileExistsResult int

const (
	FileExists FileExistsResult = iota
	FileDoesNotExist
	FilePermissionDenied
)

type BlockDevice struct {
	Path        string   `json:"path"`
	FSType      string   `json:"fstype"`
	UUID        string   `json:"uuid"`
	Label       string   `json:"label"`
	PartType    string   `json:"parttype"`
	PartUUID    string   `json:"partuuid"`
	MountPoints []string `json:"mountpoints"`
}

type API interface {
	CheckFileExists(path string) FileExistsResult
	CheckBlockDeviceExists(dev string) bool
	GetHostKernelVersion() string
	CheckUserExists(uid int) bool
	CheckGroupExists(gid int) bool
	GetBlockDevices() ([]BlockDevice, error)
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
	if strings.HasPrefix(dev, "/dev/ubd") || strings.HasPrefix(dev, "mtd:") {
		return true
	}
	resolvedPath := ResolveDevicePath(dev)
	_, err := os.Stat(resolvedPath)
	return err == nil
}

func (ActualFS) GetBlockDevices() ([]BlockDevice, error) {
	devices := make(map[string]*BlockDevice)

	// 1. Discover devices and their properties from udev
	files, _ := os.ReadDir("/run/udev/data")
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "b") {
			f, err := os.Open(filepath.Join("/run/udev/data", file.Name()))
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(f)
			parttype := ""
			fstype := ""
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "E:ID_PART_ENTRY_TYPE=") {
					parttype = strings.TrimPrefix(line, "E:ID_PART_ENTRY_TYPE=")
				}
				if strings.HasPrefix(line, "E:ID_FS_TYPE=") {
					fstype = strings.TrimPrefix(line, "E:ID_FS_TYPE=")
				}
			}
			_ = f.Close()

			majmin := strings.TrimPrefix(file.Name(), "b")
			target, _ := os.Readlink(filepath.Join("/sys/dev/block", majmin))
			if target != "" {
				devname := filepath.Base(target)
				if strings.HasPrefix(devname, "loop") || strings.HasPrefix(devname, "ram") || strings.HasPrefix(devname, "sr") {
					continue
				}
				path := "/dev/" + devname
				if devices[path] == nil {
					devices[path] = &BlockDevice{Path: path}
				}
				devices[path].PartType = parttype
				devices[path].FSType = fstype
			}
		}
	}

	// 2. Discover UUIDs
	uuids, _ := os.ReadDir("/dev/disk/by-uuid")
	for _, entry := range uuids {
		target, err := os.Readlink(filepath.Join("/dev/disk/by-uuid", entry.Name()))
		if err == nil {
			absTarget, err := filepath.Abs(filepath.Join("/dev/disk/by-uuid", target))
			if err == nil {
				if dev, ok := devices[absTarget]; ok {
					dev.UUID = entry.Name()
				}
			}
		}
	}

	// 3. Discover Labels
	labels, _ := os.ReadDir("/dev/disk/by-label")
	for _, entry := range labels {
		target, err := os.Readlink(filepath.Join("/dev/disk/by-label", entry.Name()))
		if err == nil {
			absTarget, err := filepath.Abs(filepath.Join("/dev/disk/by-label", target))
			if err == nil {
				if dev, ok := devices[absTarget]; ok {
					dev.Label = entry.Name()
				}
			}
		}
	}

	// 4. Discover PARTUUID
	partuuids, _ := os.ReadDir("/dev/disk/by-partuuid")
	for _, entry := range partuuids {
		target, err := os.Readlink(filepath.Join("/dev/disk/by-partuuid", entry.Name()))
		if err == nil {
			absTarget, err := filepath.Abs(filepath.Join("/dev/disk/by-partuuid", target))
			if err == nil {
				if dev, ok := devices[absTarget]; ok {
					dev.PartUUID = entry.Name()
				}
			}
		}
	}

	// 5. Discover Mountpoints
	if mountsData, err := os.ReadFile("/proc/mounts"); err == nil {
		lines := strings.Split(string(mountsData), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				devicePath := fields[0]
				mountPoint := fields[1]
				fsType := fields[2]
				if dev, ok := devices[devicePath]; ok {
					dev.MountPoints = append(dev.MountPoints, mountPoint)
					if dev.FSType == "" {
						dev.FSType = fsType // best effort based on mounts
					}
				} else {
					// Try to resolve if it's /dev/root
					if devicePath == "/dev/root" {
						for _, v := range devices {
							if len(v.MountPoints) == 0 {
								v.MountPoints = append(v.MountPoints, mountPoint)
								if v.FSType == "" {
									v.FSType = fsType
								}
								break
							}
						}
					}
				}
			}
		}
	}

	var result []BlockDevice
	for _, dev := range devices {
		result = append(result, *dev)
	}

	return result, nil
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

func (c *CacheFS) GetBlockDevices() ([]BlockDevice, error) {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("block_devices")); ok {
			if devs, ok := v.([]BlockDevice); ok {
				return devs, nil
			}
		}
	}
	return nil, nil
}

func (c *CacheFS) CheckBlockDeviceExists(dev string) bool {
	if strings.HasPrefix(dev, "/dev/ubd") || strings.HasPrefix(dev, "mtd:") {
		return true
	}
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
