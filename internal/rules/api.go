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
	DeviceID    string   `json:"device_id"`
}

type BlockDeviceDiagnostic struct {
	Source string
	Error  string
}

type BlockDeviceInventory struct {
	Devices     []BlockDevice
	Diagnostics []BlockDeviceDiagnostic
}

type API interface {
	CheckFileExists(path string) FileExistsResult
	CheckBlockDeviceExists(dev string) bool
	GetHostKernelVersion() string
	CheckUserExists(uid int) bool
	CheckGroupExists(gid int) bool
	GetBlockDevices() (BlockDeviceInventory, error)
	ResolveDevicePath(dev string) string
}

// OSProvider abstracts operating system operations to allow mocking
type OSProvider interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	Open(name string) (fs.File, error)
	ReadFile(name string) ([]byte, error)
	Readlink(name string) (string, error)
	Stat(name string) (fs.FileInfo, error)
}

type defaultOSProvider struct{}

func (defaultOSProvider) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }
func (defaultOSProvider) Open(name string) (fs.File, error)          { return os.Open(name) }
func (defaultOSProvider) ReadFile(name string) ([]byte, error)       { return os.ReadFile(name) }
func (defaultOSProvider) Readlink(name string) (string, error)       { return os.Readlink(name) }
func (defaultOSProvider) Stat(name string) (fs.FileInfo, error)      { return os.Stat(name) }

type ActualFS struct{
	OS OSProvider
}

func (a ActualFS) getOS() OSProvider {
	if a.OS == nil {
		return defaultOSProvider{}
	}
	return a.OS
}

func (a ActualFS) CheckUserExists(uid int) bool {
	f, err := a.getOS().Open("/etc/passwd")
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

func (a ActualFS) CheckGroupExists(gid int) bool {
	f, err := a.getOS().Open("/etc/group")
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

func (a ActualFS) CheckFileExists(path string) FileExistsResult {
	_, err := a.getOS().Stat(path)
	if err == nil {
		return FileExists
	}
	if os.IsPermission(err) {
		return FilePermissionDenied
	}
	return FileDoesNotExist
}

func (a ActualFS) CheckBlockDeviceExists(dev string) bool {
	if strings.HasPrefix(dev, "/dev/ubd") || strings.HasPrefix(dev, "mtd:") {
		return true
	}
	resolvedPath := a.ResolveDevicePath(dev)
	_, err := a.getOS().Stat(resolvedPath)
	return err == nil
}

func (a ActualFS) ResolveDevicePath(dev string) string {
	inv, err := a.GetBlockDevices()
	if err == nil {
		if matched := findDeviceByPathOrAttr(inv.Devices, dev); matched != nil {
			return matched.Path
		}
	}
	return ResolveDevicePath(dev) // fallback to util.go behavior
}

func (a ActualFS) GetBlockDevices() (BlockDeviceInventory, error) {
	osProv := a.getOS()
	devices := make(map[string]*BlockDevice)
	var diagnostics []BlockDeviceDiagnostic

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
	files, err := osProv.ReadDir("/run/udev/data")
	if err == nil {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "b") {
				f, err := osProv.Open(filepath.Join("/run/udev/data", file.Name()))
				if err != nil {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data/" + file.Name(), Error: err.Error()})
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
				if err := scanner.Err(); err != nil {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data/" + file.Name(), Error: "scanner error: " + err.Error()})
				}
				_ = f.Close()

				majmin := strings.TrimPrefix(file.Name(), "b")
				target, err := osProv.Readlink(filepath.Join("/sys/dev/block", majmin))
				if err == nil && target != "" {
					devname := filepath.Base(target)
					if strings.HasPrefix(devname, "loop") || strings.HasPrefix(devname, "ram") || strings.HasPrefix(devname, "sr") {
						continue
					}
					path := "/dev/" + devname
					dev := getOrCreate(path)
					dev.SysfsName = devname
					dev.DeviceID = majmin
					if parttype != "" {
						dev.PartType = parttype
					}
					if fstype != "" {
						dev.FSType = fstype
					}
				} else if err != nil {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/sys/dev/block/" + majmin, Error: "readlink failed: " + err.Error()})
				}
			}
		}
	} else {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data", Error: err.Error()})
	}

	// Fallback to purely sysfs block enumeration if udev failed
	if len(devices) == 0 {
		entries, err := osProv.ReadDir("/sys/class/block")
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
					continue
				}
				path := "/dev/" + name
				dev := getOrCreate(path)
				dev.SysfsName = name

				// Try to get dev id
				if b, err := osProv.ReadFile(filepath.Join("/sys/class/block", name, "dev")); err == nil {
					dev.DeviceID = strings.TrimSpace(string(b))
				}
			}
		} else {
			diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/sys/class/block", Error: err.Error()})
		}
	}

	// Helpers for symlink resolution
	resolveAliasDir := func(dir string, setFunc func(dev *BlockDevice, name string)) {
		entries, err := osProv.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: dir, Error: err.Error()})
			}
			return
		}
		for _, entry := range entries {
			target, err := osProv.Readlink(filepath.Join(dir, entry.Name()))
			if err == nil {
				// Avoid filepath.Abs which touches real host fs. We know the symlink structure.
				absTarget := filepath.Clean(filepath.Join(dir, target))
				if err == nil {
					// We only enrich devices we already know about
					if dev, ok := devices[absTarget]; ok {
						setFunc(dev, entry.Name())
					}
				} else {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: filepath.Join(dir, target), Error: "abs failed: " + err.Error()})
				}
			} else {
				diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: filepath.Join(dir, entry.Name()), Error: "readlink failed: " + err.Error()})
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
	if mountsData, err := osProv.ReadFile("/proc/mounts"); err == nil {
		lines := strings.Split(string(mountsData), "\n")
		// Need to resolve aliases first
		var deviceList []BlockDevice
		for _, dev := range devices {
			deviceList = append(deviceList, *dev)
		}
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				devicePath := fields[0]
				mountPoint := fields[1]
				fsType := fields[2]

				// Attempt to resolve through the same canonical resolution
				if matched := findDeviceByPathOrAttr(deviceList, devicePath); matched != nil {
					if dev, ok := devices[matched.Path]; ok {
						dev.MountPoints = append(dev.MountPoints, mountPoint)
						if dev.FSType == "" {
							dev.FSType = fsType // fallback best effort
						}
					}
				}
			}
		}
	} else {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/proc/mounts", Error: err.Error()})
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

	return BlockDeviceInventory{
		Devices:     result,
		Diagnostics: diagnostics,
	}, nil
}

func (a ActualFS) GetHostKernelVersion() string {
	b, err := a.getOS().ReadFile("/proc/version")
	if err == nil {
		return string(b)
	}
	b, err = a.getOS().ReadFile("/proc/sys/kernel/osrelease")
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

func (c *CacheFS) GetBlockDevices() (BlockDeviceInventory, error) {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("block_devices")); ok {
			if devs, ok := v.([]BlockDevice); ok {
				return BlockDeviceInventory{Devices: devs}, nil
			}
		}
	}
	return BlockDeviceInventory{}, nil
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
	inv, err := c.GetBlockDevices()
	if err == nil {
		if matched := findDeviceByPathOrAttr(inv.Devices, dev); matched != nil {
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
