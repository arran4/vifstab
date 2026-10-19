package txtar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/core"
	x_txtar "golang.org/x/tools/txtar"
)

type Snapshot struct {
	Fstab     []byte
	Cache     map[string]string
	MockFS    fstest.MapFS
	KernelVer string
}

func sanitizeDevice(dev *rules.BlockDevice) {
	if dev.UUID != "" {
		dev.UUID = "<sanitized>"
	}
	if dev.Label != "" {
		dev.Label = "<sanitized>"
	}
	if dev.PartUUID != "" {
		dev.PartUUID = "<sanitized>"
	}
	var cleanAliases []string
	for _, a := range dev.Aliases {
		if strings.HasPrefix(a, "/dev/disk/by-uuid/") ||
			strings.HasPrefix(a, "/dev/disk/by-partuuid/") ||
			strings.HasPrefix(a, "/dev/disk/by-label/") ||
			strings.HasPrefix(a, "/dev/disk/by-partlabel/") ||
			strings.HasPrefix(a, "/dev/disk/by-id/") {
			continue
		}
		cleanAliases = append(cleanAliases, a)
	}
	dev.Aliases = cleanAliases
}

// CaptureSystem captures the current system state, reading fstab from the
// provided path, and serializing it along with block device info and other
// environment states into a txtar byte slice.
func CaptureSystem(ctx context.Context, api rules.API, fstabPath string, unredacted bool) ([]byte, error) {
	ar := &x_txtar.Archive{}

	// Capture fstab
	fstabBytes, err := os.ReadFile(fstabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fstab from %s: %w", fstabPath, err)
	}
	ar.Files = append(ar.Files, x_txtar.File{Name: "input/etc/fstab", Data: fstabBytes})

	// Capture block devices
	inv, err := api.GetBlockDevices()
	if err != nil {
		b, _ := json.Marshal(rules.BlockDeviceInventory{Available: false})
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "provider/inventory.json",
			Data: b,
		})
	} else {
		// Serialize full struct instead of just Devices
		clonedInv := rules.BlockDeviceInventory{
			Available:   inv.Available,
			Diagnostics: inv.Diagnostics,
			Devices:     make([]rules.BlockDevice, len(inv.Devices)),
		}
		copy(clonedInv.Devices, inv.Devices)

		if !unredacted {
			for i := range clonedInv.Devices {
				sanitizeDevice(&clonedInv.Devices[i])
			}
		}

		b, err := json.Marshal(clonedInv)
		if err == nil {
			ar.Files = append(ar.Files, x_txtar.File{
				Name: "provider/inventory.json",
				Data: b,
			})
		}
	}

	// Capture kernel version
	kv := api.GetHostKernelVersion()
	if kv != "" {
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "kernel",
			Data: []byte(strings.TrimPrefix(kv, "Linux version ")),
		})
	}

	// Optionally we could probe proc_filesystems by checking /proc/filesystems
	procFsBytes, err := os.ReadFile("/proc/filesystems")
	if err == nil {
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "input/proc/filesystems",
			Data: procFsBytes,
		})
	}

	// Capture any paths referenced in fstab to see if they exist
	entries, err := core.ParseFstab(strings.NewReader(string(fstabBytes)))
	if err == nil {
		var cacheLines []string
		for _, e := range entries {
			if len(e.Fields) < 2 || strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") {
				continue
			}
			target := e.Fields[1]
			spec := e.Fields[0]
			// Check file existence
			if strings.HasPrefix(target, "/") {
				st := api.CheckFileExists(target)
				var stateStr string
				switch st {
				case rules.FileExists:
					stateStr = "true"
				case rules.FileDoesNotExist:
					stateStr = "false"
				case rules.FilePermissionDenied:
					stateStr = "permission_denied"
				default:
					stateStr = "unknown"
				}
				cacheLines = append(cacheLines, fmt.Sprintf("file_exists:%s=%s", target, stateStr))
			}

			// Resolve block device
			st := api.CheckBlockDeviceExists(spec)
			var stateStr string
			switch st {
			case rules.Exists:
				stateStr = "true"
			case rules.DoesNotExist:
				stateStr = "false"
			case rules.PermissionDenied:
				stateStr = "permission_denied"
			default:
				stateStr = "unknown"
			}
			resolved := api.ResolveDevicePath(spec)

			cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", resolved, stateStr))
			if spec != resolved {
				cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", spec, stateStr))
			}
		}

		if len(cacheLines) > 0 {
			found := false
			for i, file := range ar.Files {
				if file.Name == "cache" {
					ar.Files[i].Data = append(ar.Files[i].Data, []byte(strings.Join(cacheLines, "\n")+"\n")...)
					found = true
					break
				}
			}
			if !found {
				ar.Files = append(ar.Files, x_txtar.File{
					Name: "cache",
					Data: []byte(strings.Join(cacheLines, "\n") + "\n"),
				})
			}
		}
	}

	return x_txtar.Format(ar), nil
}

// ReplaySnapshot wraps the snapshot with a mocked rules.API implementation.
func ReplaySnapshot(snapshot *Snapshot) (rules.API, error) {
	c := rules.NewCache()
	if snapshot.KernelVer != "" {
		c.Set("proc_version", snapshot.KernelVer)
	}

	for k, v := range snapshot.Cache {
		if k == "proc_filesystems" {
			fsMap := make(map[string]bool)
			for _, item := range strings.Split(v, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					fsMap[item] = true
				}
			}
			c.Set(rules.CacheKey(k), fsMap)
		} else if k == "block_devices" {
			var inv rules.BlockDeviceInventory
			if err := json.Unmarshal([]byte(v), &inv); err == nil && (len(inv.Devices) > 0 || inv.Available) {
				c.Set(rules.CacheKey(k), inv)
			} else {
				var devs []rules.BlockDevice
				if err := json.Unmarshal([]byte(v), &devs); err == nil {
					c.Set(rules.CacheKey(k), rules.BlockDeviceInventory{Devices: devs, Available: true})
				}
			}
		} else if strings.HasPrefix(k, "file_exists:") {
			if v == "true" {
				c.Set(rules.CacheKey(k), rules.FileExists)
			} else if v == "false" {
				c.Set(rules.CacheKey(k), rules.FileDoesNotExist)
			} else if v == "permission_denied" {
				c.Set(rules.CacheKey(k), rules.FilePermissionDenied)
			} else {
				c.Set(rules.CacheKey(k), rules.Unknown)
			}
		} else if strings.HasPrefix(k, "block_exists:") {
			// For backward compatibility, the CheckBlockDeviceExists looks at file_exists:
			// Let's set both or modify CheckBlockDeviceExists. It looks at file_exists:<path>.
			if v == "true" {
				c.Set(rules.CacheKey("file_exists:"+strings.TrimPrefix(k, "block_exists:")), rules.Exists)
			} else if v == "false" {
				c.Set(rules.CacheKey("file_exists:"+strings.TrimPrefix(k, "block_exists:")), rules.DoesNotExist)
			} else if v == "permission_denied" {
				c.Set(rules.CacheKey("file_exists:"+strings.TrimPrefix(k, "block_exists:")), rules.PermissionDenied)
			} else {
				c.Set(rules.CacheKey("file_exists:"+strings.TrimPrefix(k, "block_exists:")), rules.Unknown)
			}
		} else if v == "true" {
			c.Set(rules.CacheKey(k), true)
		} else if v == "false" {
			c.Set(rules.CacheKey(k), false)
		} else {
			c.Set(rules.CacheKey(k), v)
		}
	}

	mockFs := snapshot.MockFS
	if mockFs == nil {
		mockFs = fstest.MapFS{}
	}

	if invData, ok := mockFs["provider/inventory.json"]; ok {
		var inv rules.BlockDeviceInventory
		if err := json.Unmarshal(invData.Data, &inv); err == nil {
			c.Set(rules.CacheKey("block_devices"), inv)
		}
	}

	// Add fstab into mock FS if it's missing but present in snapshot
	if snapshot.Fstab != nil {
		if _, ok := mockFs["etc/fstab"]; !ok {
			mockFs["etc/fstab"] = &fstest.MapFile{Data: snapshot.Fstab}
		}
	}

	cacheFS := rules.NewCacheFS(c, mockFs)
	cacheFS.OfflineReplay = true

	return cacheFS, nil
}

const MaxArchiveSize = 10 * 1024 * 1024 // 10 MB
const MaxSectionSize = 2 * 1024 * 1024  // 2 MB
const MaxSectionCount = 100
const MaxNameLength = 1024

func ParseTXTAR(data []byte) (*Snapshot, error) {
	if len(data) > MaxArchiveSize {
		return nil, fmt.Errorf("archive size %d exceeds limit of %d", len(data), MaxArchiveSize)
	}
	ar := x_txtar.Parse(data)
	if len(ar.Files) > MaxSectionCount {
		return nil, fmt.Errorf("archive exceeds maximum section count")
	}
	s := &Snapshot{
		Cache:  make(map[string]string),
		MockFS: fstest.MapFS{},
	}

	seenFiles := make(map[string]bool)
	for _, file := range ar.Files {
		if seenFiles[file.Name] && file.Name != "cache" {
			return nil, fmt.Errorf("duplicate file in archive: %s", file.Name)
		}
		seenFiles[file.Name] = true

		name := file.Name

		// Validation
		if strings.Contains(name, "..") {
			return nil, fmt.Errorf("unsafe path in archive: %s", name)
		}

		switch {
		case name == "provider/inventory.json":
			s.MockFS["provider/inventory.json"] = &fstest.MapFile{Data: file.Data}
		case name == "input" || name == "input/etc/fstab":

			s.Fstab = file.Data
		case name == "kernel":
			s.KernelVer = "Linux version " + strings.TrimSpace(string(file.Data))
		case name == "cache":
			for _, line := range strings.Split(string(file.Data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					s.Cache[parts[0]] = parts[1]
				}
			}
		case strings.HasPrefix(name, "input/"):
			s.MockFS[strings.TrimPrefix(name, "input/")] = &fstest.MapFile{
				Data: file.Data,
			}
		}
	}

	return s, nil
}
