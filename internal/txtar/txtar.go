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

type Manifest struct {
	Version  string `json:"version"`
	Type     string `json:"type"`
	Redacted *bool  `json:"redacted"`
}

type Snapshot struct {
	Fstab     []byte
	Cache     map[string]string
	MockFS    fstest.MapFS
	KernelVer string
	Manifest  *Manifest
}

func sanitizeStr(val string, prefix string, redactedMap map[string]string) string {
	if val == "" {
		return ""
	}
	if mapped, ok := redactedMap[val]; ok {
		return mapped
	}
	var newVal string
	if prefix == "uuid" || prefix == "partuuid" {
		// Keep it looking like a UUID so parsing doesn't break
		newVal = fmt.Sprintf("00000000-0000-0000-0000-%012d", len(redactedMap))
	} else {
		newVal = fmt.Sprintf("<sanitized-%s-%d>", prefix, len(redactedMap))
	}
	redactedMap[val] = newVal
	return newVal
}

func sanitizeDevice(dev *rules.BlockDevice, redactedMap map[string]string) {
	if dev.UUID != "" {
		newVal := sanitizeStr(dev.UUID, "uuid", redactedMap)
		redactedMap["UUID="+dev.UUID] = "UUID=" + newVal
		dev.UUID = newVal
	}
	if dev.Label != "" {
		newVal := sanitizeStr(dev.Label, "label", redactedMap)
		redactedMap["LABEL="+dev.Label] = "LABEL=" + newVal
		dev.Label = newVal
	}
	if dev.PartUUID != "" {
		newVal := sanitizeStr(dev.PartUUID, "partuuid", redactedMap)
		redactedMap["PARTUUID="+dev.PartUUID] = "PARTUUID=" + newVal
		dev.PartUUID = newVal
	}
	var cleanAliases []string
	for _, a := range dev.Aliases {
		if strings.HasPrefix(a, "/dev/disk/by-uuid/") {
			cleanAliases = append(cleanAliases, "/dev/disk/by-uuid/"+sanitizeStr(strings.TrimPrefix(a, "/dev/disk/by-uuid/"), "uuid", redactedMap))
		} else if strings.HasPrefix(a, "/dev/disk/by-partuuid/") {
			cleanAliases = append(cleanAliases, "/dev/disk/by-partuuid/"+sanitizeStr(strings.TrimPrefix(a, "/dev/disk/by-partuuid/"), "partuuid", redactedMap))
		} else if strings.HasPrefix(a, "/dev/disk/by-label/") {
			cleanAliases = append(cleanAliases, "/dev/disk/by-label/"+sanitizeStr(strings.TrimPrefix(a, "/dev/disk/by-label/"), "label", redactedMap))
		} else if strings.HasPrefix(a, "/dev/disk/by-partlabel/") {
			cleanAliases = append(cleanAliases, "/dev/disk/by-partlabel/"+sanitizeStr(strings.TrimPrefix(a, "/dev/disk/by-partlabel/"), "label", redactedMap))
		} else if strings.HasPrefix(a, "/dev/disk/by-id/") || strings.HasPrefix(a, "/dev/disk/by-path/") {
			// Don't keep raw by-id/by-path since they contain serials/topology
			continue
		} else {
			// Just map it as a path
			newVal := "/sanitized-" + sanitizeStr(a, "path", redactedMap)
			redactedMap[a] = newVal
			cleanAliases = append(cleanAliases, newVal)
		}
	}
	dev.Aliases = cleanAliases
}

// CaptureSystem captures the current system state.
func CaptureSystem(ctx context.Context, api rules.API, fstabPath string, unredacted bool) ([]byte, error) {
	ar := &x_txtar.Archive{}
	redactedMap := make(map[string]string)

	fstabBytes, err := os.ReadFile(fstabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fstab from %s: %w", fstabPath, err)
	}
	fstabStr := string(fstabBytes)

	entries, err := core.ParseFstab(strings.NewReader(fstabStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse fstab for capture: %w", err)
	}

	var cacheLines []string

	// Capture block devices
	inv, err := api.GetBlockDevices()
	if err != nil {
		diagMsg := err.Error()
		if !unredacted {
			diagMsg = "<sanitized-error>"
		}
		b, _ := json.Marshal(rules.BlockDeviceInventory{Available: false, Diagnostics: []rules.BlockDeviceDiagnostic{{Error: diagMsg}}})
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "provider/inventory.json",
			Data: b,
		})
	} else {
		clonedInv := rules.BlockDeviceInventory{
			Available:   inv.Available,
			Diagnostics: inv.Diagnostics,
			Devices:     make([]rules.BlockDevice, len(inv.Devices)),
		}
		if !unredacted {
			clonedInv.Diagnostics = nil // clear diagnostics to prevent leaks
		}
		copy(clonedInv.Devices, inv.Devices)

		if !unredacted {
			for i := range clonedInv.Devices {
				// We clear host-identifying device details like DeviceID and SysfsName which might contain hardware serials
				clonedInv.Devices[i].DeviceID = ""
				clonedInv.Devices[i].SysfsName = ""

				// Path (like /dev/sda1) is required for resolution.
				// We map it via redactedMap so it stays resolvable (and is consistently rewritten everywhere).
				if clonedInv.Devices[i].Path != "" {
					sanitizedPath := "/sanitized-" + sanitizeStr(clonedInv.Devices[i].Path, "path", redactedMap)
					redactedMap[clonedInv.Devices[i].Path] = sanitizedPath
					clonedInv.Devices[i].Path = sanitizedPath
				}

				// Clear MountPoints since it gives away other parts of the system topology
				clonedInv.Devices[i].MountPoints = nil
				sanitizeDevice(&clonedInv.Devices[i], redactedMap)
			}
		}

		b, err := json.Marshal(clonedInv)
		if err == nil {
			ar.Files = append(ar.Files, x_txtar.File{
				Name: "provider/inventory.json",
				Data: b,
			})
		} else {
			return nil, fmt.Errorf("failed to marshal block devices: %w", err)
		}
	}

	var outStr string

	for _, e := range entries {
		if strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") || len(e.Fields) < 2 {
			line := e.RawLine
			if !unredacted {
				if strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") {
					line = "# <sanitized-comment>"
				} else if len(e.Fields) == 1 {
					line = "<sanitized-malformed-line>"
				}
			}
			outStr += line + "\n"
			continue
		}

		origSpec := e.Fields[0]
		origTarget := e.Fields[1]

		outSpec := origSpec
		outTarget := origTarget
		outLineFields := make([]string, len(e.Fields))
		copy(outLineFields, e.Fields)

		if !unredacted {
			// Redact spec safely maintaining structure
			if strings.HasPrefix(origSpec, "UUID=") {
				val := strings.TrimPrefix(origSpec, "UUID=")
				newVal := sanitizeStr(val, "uuid", redactedMap)
				outSpec = "UUID=" + newVal
			} else if strings.HasPrefix(origSpec, "LABEL=") {
				val := strings.TrimPrefix(origSpec, "LABEL=")
				newVal := sanitizeStr(val, "label", redactedMap)
				outSpec = "LABEL=" + newVal
			} else if strings.HasPrefix(origSpec, "PARTUUID=") {
				val := strings.TrimPrefix(origSpec, "PARTUUID=")
				newVal := sanitizeStr(val, "partuuid", redactedMap)
				outSpec = "PARTUUID=" + newVal
			} else if strings.HasPrefix(origSpec, "//") {
				// CIFS
				val := sanitizeStr(origSpec, "net", redactedMap)
				outSpec = "//" + strings.Trim(val, "<>")
			} else if strings.Contains(origSpec, ":/") {
				// NFS
				val := sanitizeStr(origSpec, "net", redactedMap)
				outSpec = strings.Trim(val, "<>") + ":/export"
			} else if strings.HasPrefix(origSpec, "/") {
				if !strings.HasPrefix(origSpec, "/dev/") {
					val := sanitizeStr(origSpec, "path", redactedMap)
					outSpec = "/" + strings.Trim(val, "<>")
				}
			}

			// Redact target safely
			if strings.HasPrefix(origTarget, "/") {
				val := sanitizeStr(origTarget, "path", redactedMap)
				// val is already <sanitized-path-N>, so Trim removes <> making it sanitized-path-N
				outTarget = "/" + strings.Trim(val, "<>")
			}

			outLineFields[0] = outSpec
			outLineFields[1] = outTarget

			// Redact options if any
			if len(outLineFields) >= 4 {
				opts := outLineFields[3]
				var newOpts []string
				for _, opt := range strings.Split(opts, ",") {
					if strings.HasPrefix(opt, "password=") || strings.HasPrefix(opt, "username=") || strings.HasPrefix(opt, "credentials=") || strings.HasPrefix(opt, "secret=") {
						parts := strings.SplitN(opt, "=", 2)
						if len(parts) == 2 {
							sanitizedVal := sanitizeStr(parts[1], "secret", redactedMap)
							newOpts = append(newOpts, parts[0]+"="+sanitizedVal)
						} else {
							newOpts = append(newOpts, opt)
						}
					} else {
						newOpts = append(newOpts, opt)
					}
				}
				outLineFields[3] = strings.Join(newOpts, ",")
			}
		}

		// Build cache using original queries but recording under outSpec/outTarget keys
		if strings.HasPrefix(origTarget, "/") {
			st := api.CheckFileExists(origTarget)
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
			cacheLines = append(cacheLines, fmt.Sprintf("file_exists:%s=%s", outTarget, stateStr))
		}

		st := api.CheckBlockDeviceExists(origSpec)
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

		origResolved := api.ResolveDevicePath(origSpec)
		outResolved := origResolved

		if !unredacted {
			// If the resolved target is different, it needs to be redacted too before caching.
			if mapped, ok := redactedMap[origResolved]; ok {
				outResolved = mapped
			} else if origSpec != origResolved {
				if strings.HasPrefix(origResolved, "/dev/disk/by-uuid/") {
					outResolved = "/dev/disk/by-uuid/" + sanitizeStr(strings.TrimPrefix(origResolved, "/dev/disk/by-uuid/"), "uuid", redactedMap)
				} else if strings.HasPrefix(origResolved, "/dev/disk/by-partuuid/") {
					outResolved = "/dev/disk/by-partuuid/" + sanitizeStr(strings.TrimPrefix(origResolved, "/dev/disk/by-partuuid/"), "partuuid", redactedMap)
				} else if strings.HasPrefix(origResolved, "/dev/disk/by-label/") {
					outResolved = "/dev/disk/by-label/" + sanitizeStr(strings.TrimPrefix(origResolved, "/dev/disk/by-label/"), "label", redactedMap)
				} else if strings.HasPrefix(origResolved, "/") {
					// Could be /dev/sda1 (which might not be in inventory if unavailable)
					// Or could be /mnt/net
					outResolved = "/sanitized-" + sanitizeStr(origResolved, "path", redactedMap)
				} else if strings.HasPrefix(origResolved, "//") || strings.Contains(origResolved, ":/") {
					// Network path
					val := sanitizeStr(origResolved, "net", redactedMap)
					outResolved = "//" + strings.Trim(val, "<>")
				}
			}
		}

		cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", outResolved, stateStr))
		if outSpec != outResolved {
			cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", outSpec, stateStr))
		}

		// Render fstab line deterministically
		if unredacted {
			outStr += e.RawLine + "\n"
		} else {
			var outFields []string
			for i, f := range e.Fields {
				if i < len(outLineFields) {
					outFields = append(outFields, outLineFields[i])
				} else {
					outFields = append(outFields, f)
				}
			}
			outStr += strings.Join(outFields, " ") + "\n"
		}
	}

	manifest := map[string]interface{}{
		"version":  "1.0",
		"type":     "system",
		"redacted": !unredacted,
	}
	if b, err := json.Marshal(manifest); err == nil {
		ar.Files = append(ar.Files, x_txtar.File{Name: "manifest.json", Data: append(b, '\n')})
	}
	ar.Files = append(ar.Files, x_txtar.File{Name: "input/etc/fstab", Data: []byte(outStr)})

	kv := api.GetHostKernelVersion()
	if kv != "" {
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "kernel",
			Data: []byte(strings.TrimPrefix(kv, "Linux version ")),
		})
	}

	// Only capture /proc/filesystems if it's available in the cache/context (which rules.API might expose indirectly).
	// For now we don't directly read os.ReadFile("/proc/filesystems") during capture because it breaks isolation mocks.

	if len(cacheLines) > 0 {
		ar.Files = append(ar.Files, x_txtar.File{
			Name: "cache",
			Data: []byte(strings.Join(cacheLines, "\n") + "\n"),
		})
	}

	return x_txtar.Format(ar), nil
}

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
		} else {
			return nil, fmt.Errorf("malformed provider/inventory.json: %w", err)
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
		if len(name) > MaxNameLength {
			return nil, fmt.Errorf("archive file name exceeds maximum length")
		}
		if len(file.Data) > MaxSectionSize {
			return nil, fmt.Errorf("archive file %s exceeds maximum section size", name)
		}

		switch {
		case name == "manifest.json":
			var manifest Manifest
			if err := json.Unmarshal(file.Data, &manifest); err != nil {
				return nil, fmt.Errorf("malformed manifest.json: %w", err)
			}

			if manifest.Redacted == nil {
				return nil, fmt.Errorf("manifest missing required field 'redacted'")
			}
			if manifest.Version != "1.0" {
				return nil, fmt.Errorf("unsupported manifest version: %v", manifest.Version)
			}
			if manifest.Type != "system" {
				return nil, fmt.Errorf("unsupported manifest type: %v", manifest.Type)
			}
			s.Manifest = &manifest
		case name == "provider/inventory.json":
			s.MockFS["provider/inventory.json"] = &fstest.MapFile{Data: file.Data}
		case name == "input" || name == "input/etc/fstab":
			if s.Fstab != nil {
				return nil, fmt.Errorf("duplicate or ambiguous fstab section in archive")
			}
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
		case strings.HasPrefix(name, "input/") && name != "input/etc/fstab":
			s.MockFS[strings.TrimPrefix(name, "input/")] = &fstest.MapFile{
				Data: file.Data,
			}
		}
	}

	if s.Fstab == nil {
		return nil, fmt.Errorf("invalid archive: missing fstab data")
	}

	return s, nil
}
