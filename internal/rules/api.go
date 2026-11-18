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

type ExistState int

const (
	Exists ExistState = iota
	DoesNotExist
	PermissionDenied
	Unknown
)

type FileExistsResult = ExistState

const (
	FileExists           FileExistsResult = Exists
	FileDoesNotExist     FileExistsResult = DoesNotExist
	FilePermissionDenied FileExistsResult = PermissionDenied
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
	Aliases     []string `json:"aliases"`
}

type BlockDeviceDiagnostic struct {
	Source string
	Error  string
}

type BlockDeviceInventory struct {
	Devices     []BlockDevice
	Diagnostics []BlockDeviceDiagnostic
	Available   bool
}

func (inv BlockDeviceInventory) Resolve(ref string) *BlockDevice {
	return findDeviceByPathOrAttr(inv.Devices, ref)
}

type API interface {
	CheckFileExists(path string) FileExistsResult
	CheckBlockDeviceExists(dev string) ExistState
	GetHostKernelVersion() string
	CheckUserExists(uid int) ExistState
	CheckGroupExists(gid int) ExistState
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

type ActualFS struct {
	OS OSProvider
}

func (a ActualFS) getOS() OSProvider {
	if a.OS == nil {
		return defaultOSProvider{}
	}
	return a.OS
}

func (a ActualFS) CheckUserExists(uid int) ExistState {
	return observeAccountFile(a.getOS().Open, "/etc/passwd", uid, true)
}

func (a ActualFS) CheckGroupExists(gid int) ExistState {
	return observeAccountFile(a.getOS().Open, "/etc/group", gid, false)
}

func (a ActualFS) CheckFileExists(path string) FileExistsResult {
	_, err := a.getOS().Stat(path)
	if err == nil {
		return FileExists
	}
	if os.IsPermission(err) {
		return FilePermissionDenied
	}
	if os.IsNotExist(err) {
		return FileDoesNotExist
	}
	return Unknown
}

func (a ActualFS) CheckBlockDeviceExists(dev string) ExistState {
	if strings.HasPrefix(dev, "/dev/ubd") || strings.HasPrefix(dev, "mtd:") {
		return Exists
	}
	resolvedPath := a.ResolveDevicePath(dev)
	_, err := a.getOS().Stat(resolvedPath)
	if err == nil {
		return Exists
	}
	if os.IsPermission(err) {
		return PermissionDenied
	}
	if os.IsNotExist(err) {
		return DoesNotExist
	}
	return Unknown
}

func (a ActualFS) ResolveDevicePath(dev string) string {
	return ResolveDevicePath(dev)
}

func (a ActualFS) GetBlockDevices() (BlockDeviceInventory, error) {
	osProv := a.getOS()
	devicesByID := make(map[string]*BlockDevice)
	var diagnostics []BlockDeviceDiagnostic
	available := false

	// Helper to get or create device by ID, falling back to path if ID unknown
	getOrCreate := func(deviceID, path string) *BlockDevice {
		if deviceID != "" {
			if dev, ok := devicesByID[deviceID]; ok {
				return dev
			}
		}
		// If we don't have ID, or we didn't find it by ID, fall back to path lookup
		for _, dev := range devicesByID {
			if dev.Path == path {
				// If we now have an ID and it didn't before, update it
				if deviceID != "" && dev.DeviceID == "" {
					dev.DeviceID = deviceID
					devicesByID[deviceID] = dev
				}
				return dev
			}
		}

		dev := &BlockDevice{Path: path, DeviceID: deviceID}
		key := deviceID
		if key == "" {
			key = path // fallback key
		}
		devicesByID[key] = dev
		return dev
	}

	// 1. Establish canonical baseline from sysfs
	entries, err := osProv.ReadDir("/sys/class/block")
	if err == nil {
		available = true
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
				continue
			}

			deviceID := ""
			if b, err := osProv.ReadFile(filepath.Join("/sys/class/block", name, "dev")); err == nil {
				deviceID = strings.TrimSpace(string(b))
			}

			path := "/dev/" + name
			dev := getOrCreate(deviceID, path)
			dev.SysfsName = name
		}
	} else {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/sys/class/block", Error: err.Error()})
	}

	// 2. Enrich from udev data
	files, err := osProv.ReadDir("/run/udev/data")
	if err == nil {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "b") {
				majmin := strings.TrimPrefix(file.Name(), "b")

				f, err := osProv.Open(filepath.Join("/run/udev/data", file.Name()))
				if err != nil {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data/" + file.Name(), Error: err.Error()})
					continue
				}
				scanner := bufio.NewScanner(f)
				parttype := ""
				fstype := ""
				devnameStr := ""
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "E:ID_PART_ENTRY_TYPE=") {
						parttype = strings.TrimPrefix(line, "E:ID_PART_ENTRY_TYPE=")
					}
					if strings.HasPrefix(line, "E:ID_FS_TYPE=") {
						fstype = strings.TrimPrefix(line, "E:ID_FS_TYPE=")
					}
					if strings.HasPrefix(line, "E:DEVNAME=") {
						devnameStr = strings.TrimPrefix(line, "E:DEVNAME=")
					}
				}
				if err := scanner.Err(); err != nil {
					diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data/" + file.Name(), Error: "scanner error: " + err.Error()})
				}
				_ = f.Close()

				path := ""
				if devnameStr != "" {
					path = devnameStr
				} else {
					target, err := osProv.Readlink(filepath.Join("/sys/dev/block", majmin))
					if err == nil && target != "" {
						path = "/dev/" + filepath.Base(target)
					} else {
						// Can't determine path, skip
						continue
					}
				}

				if strings.HasPrefix(path, "/dev/loop") || strings.HasPrefix(path, "/dev/ram") || strings.HasPrefix(path, "/dev/sr") {
					continue
				}

				dev := getOrCreate(majmin, path)
				available = true
				if parttype != "" {
					dev.PartType = parttype
				}
				if fstype != "" {
					dev.FSType = fstype
				}
			}
		}
	} else {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/run/udev/data", Error: err.Error()})
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

				// Find device by path
				for _, dev := range devicesByID {
					if dev.Path == absTarget {
						setFunc(dev, entry.Name())
						break
					}
				}
			} else {
				diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: filepath.Join(dir, entry.Name()), Error: "readlink failed: " + err.Error()})
			}
		}
	}

	// 3. Discover UUIDs
	resolveAliasDir("/dev/disk/by-uuid", func(dev *BlockDevice, name string) { dev.UUID = name })

	// 4. Discover Labels
	resolveAliasDir("/dev/disk/by-label", func(dev *BlockDevice, name string) { dev.Label = name })

	// 5. Discover PARTUUID
	resolveAliasDir("/dev/disk/by-partuuid", func(dev *BlockDevice, name string) { dev.PartUUID = name })

	// Discover generic aliases
	for _, dir := range []string{"/dev/disk/by-partlabel", "/dev/disk/by-id", "/dev/disk/by-path"} {
		resolveAliasDir(dir, func(dev *BlockDevice, name string) { dev.Aliases = append(dev.Aliases, dir+"/"+name) })
	}

	// 6. Discover Mountpoints
	if mountsData, err := osProv.ReadFile("/proc/mounts"); err == nil {
		lines := strings.Split(string(mountsData), "\n")

		var deviceList []BlockDevice
		for _, dev := range devicesByID {
			deviceList = append(deviceList, *dev)
		}
		invTemp := BlockDeviceInventory{Devices: deviceList}

		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				devicePath := fields[0]
				mountPoint := fields[1]
				fsType := fields[2]

				if matched := invTemp.Resolve(devicePath); matched != nil {
					for _, dev := range devicesByID {
						if dev.Path == matched.Path || (dev.DeviceID != "" && matched.DeviceID != "" && dev.DeviceID == matched.DeviceID) {
							dev.MountPoints = append(dev.MountPoints, mountPoint)
							if dev.FSType == "" {
								dev.FSType = fsType
							}
							break
						}
					}
				}
			}
		}
	} else {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: "/proc/mounts", Error: err.Error()})
	}

	var result []BlockDevice

	// Deduplicate by taking unique pointers
	uniqueDevs := make(map[*BlockDevice]bool)
	for _, dev := range devicesByID {
		if !uniqueDevs[dev] {
			uniqueDevs[dev] = true
			result = append(result, *dev)
		}
	}

	for i := range result {
		// Unique and sort mountpoints
		dev := &result[i]
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

		// Unique and sort aliases
		aliasMap := make(map[string]bool)
		var aliases []string
		for _, a := range dev.Aliases {
			if !aliasMap[a] {
				aliasMap[a] = true
				aliases = append(aliases, a)
			}
		}
		sort.Strings(aliases)
		dev.Aliases = aliases
	}

	// Deterministic ordering by path
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].DeviceID < result[j].DeviceID
	})

	return BlockDeviceInventory{
		Devices:     result,
		Diagnostics: diagnostics,
		Available:   available,
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
		if os.IsNotExist(err) {
			return FileDoesNotExist
		}
		return Unknown
	}
	panic("CacheFS must be initialized with a valid fs.FS")
}

func (c *CacheFS) CheckUserExists(uid int) ExistState {
	if c.Cache != nil {
		cacheKey := CacheKey("user_exists:" + strconv.Itoa(uid))
		if v, ok := c.Cache.Get(cacheKey); ok {
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
	}
	if c.FS == nil {
		return Unknown
	}
	result := observeAccountFile(c.FS.Open, "etc/passwd", uid, true)
	if c.Cache != nil {
		c.Cache.Set(CacheKey("user_exists:"+strconv.Itoa(uid)), result)
	}
	return result
}

func (c *CacheFS) CheckGroupExists(gid int) ExistState {
	if c.Cache != nil {
		cacheKey := CacheKey("group_exists:" + strconv.Itoa(gid))
		if v, ok := c.Cache.Get(cacheKey); ok {
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
	}
	if c.FS == nil {
		return Unknown
	}
	result := observeAccountFile(c.FS.Open, "etc/group", gid, false)
	if c.Cache != nil {
		c.Cache.Set(CacheKey("group_exists:"+strconv.Itoa(gid)), result)
	}
	return result
}

func (c *CacheFS) GetBlockDevices() (BlockDeviceInventory, error) {
	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("block_devices")); ok {
			if devs, ok := v.([]BlockDevice); ok {
				return BlockDeviceInventory{Devices: devs, Available: true}, nil
			}
			if inv, ok := v.(BlockDeviceInventory); ok {
				return inv, nil
			}
		}
	}
	return BlockDeviceInventory{Available: false}, nil
}

func (c *CacheFS) CheckBlockDeviceExists(dev string) ExistState {
	if strings.HasPrefix(dev, "/dev/ubd") || strings.HasPrefix(dev, "mtd:") {
		return Exists
	}
	resolvedPath := c.ResolveDevicePath(dev)

	if c.Cache != nil {
		if v, ok := c.Cache.Get(CacheKey("file_exists:" + resolvedPath)); ok {
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
		// Also check if they cached the raw path for some reason
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
	}
	if c.FS != nil {
		checkPath := resolvedPath
		if checkPath != "" && checkPath[0] == '/' {
			checkPath = checkPath[1:]
		}
		_, err := fs.Stat(c.FS, checkPath)
		if err == nil {
			return Exists
		}
		if os.IsPermission(err) {
			return PermissionDenied
		}
		if os.IsNotExist(err) {
			return DoesNotExist
		}
		return Unknown
	}
	panic("CacheFS must be initialized with a valid fs.FS")
}

func (c *CacheFS) ResolveDevicePath(dev string) string {
	return ResolveDevicePath(dev)
}

func findDeviceByPathOrAttr(devs []BlockDevice, dev string) *BlockDevice {
	if dev == "" {
		return nil
	}

	if strings.HasPrefix(dev, "UUID=") {
		uuid := strings.Trim(strings.TrimPrefix(dev, "UUID="), `"'`)
		for i := range devs {
			if strings.EqualFold(devs[i].UUID, uuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "LABEL=") {
		label := normalizeDiskReference(strings.Trim(strings.TrimPrefix(dev, "LABEL="), `"'`))
		for i := range devs {
			if normalizeDiskReference(devs[i].Label) == label {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "PARTUUID=") {
		partuuid := strings.Trim(strings.TrimPrefix(dev, "PARTUUID="), `"'`)
		for i := range devs {
			if strings.EqualFold(devs[i].PartUUID, partuuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "PARTLABEL=") {
		partlabel := normalizeDiskReference(strings.Trim(strings.TrimPrefix(dev, "PARTLABEL="), `"'`))
		for i := range devs {
			for _, alias := range devs[i].Aliases {
				if strings.HasPrefix(alias, "/dev/disk/by-partlabel/") && normalizeDiskReference(strings.TrimPrefix(alias, "/dev/disk/by-partlabel/")) == partlabel {
					return &devs[i]
				}
			}
		}
	}
	if strings.HasPrefix(dev, "ID=") {
		id := strings.Trim(strings.TrimPrefix(dev, "ID="), `"'`)
		expectedAlias := "/dev/disk/by-id/" + id
		for i := range devs {
			for _, alias := range devs[i].Aliases {
				if alias == expectedAlias {
					return &devs[i]
				}
			}
		}
	}

	if strings.HasPrefix(dev, "/dev/disk/by-uuid/") {
		uuid := strings.TrimPrefix(dev, "/dev/disk/by-uuid/")
		for i := range devs {
			if strings.EqualFold(devs[i].UUID, uuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-label/") {
		label := normalizeDiskReference(strings.TrimPrefix(dev, "/dev/disk/by-label/"))
		for i := range devs {
			if normalizeDiskReference(devs[i].Label) == label {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-partuuid/") {
		partuuid := strings.TrimPrefix(dev, "/dev/disk/by-partuuid/")
		for i := range devs {
			if strings.EqualFold(devs[i].PartUUID, partuuid) {
				return &devs[i]
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-partlabel/") {
		partlabel := normalizeDiskReference(strings.TrimPrefix(dev, "/dev/disk/by-partlabel/"))
		for i := range devs {
			for _, alias := range devs[i].Aliases {
				if strings.HasPrefix(alias, "/dev/disk/by-partlabel/") && normalizeDiskReference(strings.TrimPrefix(alias, "/dev/disk/by-partlabel/")) == partlabel {
					return &devs[i]
				}
			}
		}
	}

	for i := range devs {
		for _, alias := range devs[i].Aliases {
			if alias == dev {
				return &devs[i]
			}
		}
	}

	for i := range devs {
		if devs[i].Path == dev {
			return &devs[i]
		}
	}

	return nil
}

func normalizeDiskReference(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); {
		if value[i] == '\\' && i+3 < len(value) {
			if value[i+1] == 'x' {
				if decoded, err := strconv.ParseUint(value[i+2:i+4], 16, 8); err == nil {
					b.WriteByte(byte(decoded))
					i += 4
					continue
				}
			} else if value[i+1] >= '0' && value[i+1] <= '7' && value[i+2] >= '0' && value[i+2] <= '7' && value[i+3] >= '0' && value[i+3] <= '7' {
				decoded, _ := strconv.ParseUint(value[i+1:i+4], 8, 8)
				b.WriteByte(byte(decoded))
				i += 4
				continue
			}
		}
		b.WriteByte(value[i])
		i++
	}
	return b.String()
}
