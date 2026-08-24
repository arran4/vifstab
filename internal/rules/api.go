package rules

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	SysfsName   string   `json:"sysfs_name"`
}

type API interface {
	CheckFileExists(path string) FileExistsResult
	CheckBlockDeviceExists(dev string) bool
	GetHostKernelVersion() string
	CheckUserExists(uid int) bool
	CheckGroupExists(gid int) bool
	GetBlockDevices() ([]BlockDevice, error)
	ResolveDevicePath(dev string) string
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
	resolvedPath := ActualFS{}.ResolveDevicePath(dev)
	_, err := os.Stat(resolvedPath)
	return err == nil
}

func (ActualFS) ResolveDevicePath(dev string) string {
	return ResolveDevicePath(dev) // fallback to util.go behavior
}

func (ActualFS) GetBlockDevices() ([]BlockDevice, error) {
	devices := make(map[string]*BlockDevice)
	var anyErr error

	// Helper to get or create device
	getOrCreate := func(path string) *BlockDevice {
		if dev, ok := devices[path]; ok {
			return dev
		}
		dev := &BlockDevice{Path: path}
		devices[path] = dev
		return dev
	}

	// 1. Discover devices and their properties from udev
	files, err := os.ReadDir("/run/udev/data")
	if err == nil {
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
				target, err := os.Readlink(filepath.Join("/sys/dev/block", majmin))
				if err == nil && target != "" {
					devname := filepath.Base(target)
					if strings.HasPrefix(devname, "loop") || strings.HasPrefix(devname, "ram") || strings.HasPrefix(devname, "sr") {
						continue
					}
					path := "/dev/" + devname
					dev := getOrCreate(path)
					dev.SysfsName = devname
					if parttype != "" {
						dev.PartType = parttype
					}
					if fstype != "" {
						dev.FSType = fstype
					}
				}
			}
		}
	} else {
		anyErr = err
	}

	// Fallback to purely sysfs block enumeration if udev failed
	if len(devices) == 0 {
		entries, err := os.ReadDir("/sys/class/block")
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
					continue
				}
				path := "/dev/" + name
				dev := getOrCreate(path)
				dev.SysfsName = name
			}
		}
	}

	// Helpers for symlink resolution
	resolveAliasDir := func(dir string, setFunc func(dev *BlockDevice, name string)) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				anyErr = err
			}
			return
		}
		for _, entry := range entries {
			target, err := os.Readlink(filepath.Join(dir, entry.Name()))
			if err == nil {
				absTarget, err := filepath.Abs(filepath.Join(dir, target))
				if err == nil {
					// We only enrich devices we already know about
					if dev, ok := devices[absTarget]; ok {
						setFunc(dev, entry.Name())
					}
				}
			}
		}
	}

	// 2. Discover UUIDs
	resolveAliasDir("/dev/disk/by-uuid", func(dev *BlockDevice, name string) { dev.UUID = name })

	// 3. Discover Labels
	resolveAliasDir("/dev/disk/by-label", func(dev *BlockDevice, name string) { dev.Label = name })

	// 4. Discover PARTUUID
	resolveAliasDir("/dev/disk/by-partuuid", func(dev *BlockDevice, name string) { dev.PartUUID = name })

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
						dev.FSType = fsType // fallback best effort
					}
				} else {
					// Deterministic /dev/root fallback
					if devicePath == "/dev/root" {
						for _, v := range devices {
							// If we found a block device with the same fstype and no mountpoints
							if len(v.MountPoints) == 0 && (v.FSType == fsType || v.FSType == "") {
								// We cannot reliably guess without more info, so do not arbitrarily assign.
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
		// Unique and sort mountpoints
		mpMap := make(map[string]bool)
		var mps []string
		for _, m := range dev.MountPoints {
			if !mpMap[m] {
				mpMap[m] = true
				mps = append(mps, m)
			}
		}
		sort.Strings(mps)
		dev.MountPoints = mps

		result = append(result, *dev)
	}

	// Deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result, anyErr
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
	resolvedPath := c.ResolveDevicePath(dev)

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

func (c *CacheFS) ResolveDevicePath(dev string) string {
	devs, err := c.GetBlockDevices()
	if err == nil {
		if matched := findDeviceByPathOrAttr(devs, dev); matched != nil {
			return matched.Path
		}
	}
	return ResolveDevicePath(dev)
}

func findDeviceByPathOrAttr(devs []BlockDevice, dev string) *BlockDevice {
	if strings.HasPrefix(dev, "UUID=") {
		uuid := strings.Trim(strings.TrimPrefix(dev, "UUID="), `"'`)
		for i, d := range devs {
			if strings.EqualFold(d.UUID, uuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "LABEL=") {
		label := strings.Trim(strings.TrimPrefix(dev, "LABEL="), `"'`)
		for i, d := range devs {
			if d.Label == label {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "PARTUUID=") {
		partuuid := strings.Trim(strings.TrimPrefix(dev, "PARTUUID="), `"'`)
		for i, d := range devs {
			if strings.EqualFold(d.PartUUID, partuuid) {
				return &devs[i]
			}
		}
	}

	if strings.HasPrefix(dev, "/dev/disk/by-uuid/") {
		uuid := strings.TrimPrefix(dev, "/dev/disk/by-uuid/")
		for i, d := range devs {
			if strings.EqualFold(d.UUID, uuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-label/") {
		label := strings.TrimPrefix(dev, "/dev/disk/by-label/")
		for i, d := range devs {
			if d.Label == label {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-partuuid/") {
		partuuid := strings.TrimPrefix(dev, "/dev/disk/by-partuuid/")
		for i, d := range devs {
			if strings.EqualFold(d.PartUUID, partuuid) {
				return &devs[i]
			}
		}
	}

	for i, d := range devs {
		if d.Path == dev {
			return &devs[i]
		}
	}

	evaled, err := filepath.EvalSymlinks(dev)
	if err == nil {
		for i, d := range devs {
			if d.Path == evaled {
				return &devs[i]
			}
		}
	}

	return nil
}
