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

func sanitizeStr(val string, prefix string, redactedMap map[string]string) string {
	if val == "" {
		return ""
	}
	if mapped, ok := redactedMap[val]; ok {
		return mapped
	}
	newVal := fmt.Sprintf("<sanitized-%s-%d>", prefix, len(redactedMap))
	redactedMap[val] = newVal
	return newVal
}

func sanitizeDevice(dev *rules.BlockDevice, redactedMap map[string]string) {
	if dev.UUID != "" {
		dev.UUID = sanitizeStr(dev.UUID, "uuid", redactedMap)
	}
	if dev.Label != "" {
		dev.Label = sanitizeStr(dev.Label, "label", redactedMap)
	}
	if dev.PartUUID != "" {
		dev.PartUUID = sanitizeStr(dev.PartUUID, "partuuid", redactedMap)
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
		} else if strings.HasPrefix(a, "/dev/disk/by-id/") {
			cleanAliases = append(cleanAliases, "/dev/disk/by-id/"+sanitizeStr(strings.TrimPrefix(a, "/dev/disk/by-id/"), "id", redactedMap))
		} else {
			cleanAliases = append(cleanAliases, a)
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
		b, _ := json.Marshal(rules.BlockDeviceInventory{Available: false, Diagnostics: []rules.BlockDeviceDiagnostic{{Error: err.Error()}}})
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
		copy(clonedInv.Devices, inv.Devices)

		if !unredacted {
			for i := range clonedInv.Devices {
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
		    // Fail explicitly if marshal fails
		    return nil, fmt.Errorf("failed to marshal block devices: %w", err)
		}
	}

	for _, e := range entries {
		if len(e.Fields) < 2 || strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") || e.Fields[1] == "" {
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
			if !unredacted {
				for old, new := range redactedMap {
					target = strings.ReplaceAll(target, old, new)
				}
			}
			cacheLines = append(cacheLines, fmt.Sprintf("file_exists:%s=%s", target, stateStr))
		}

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
		if !unredacted {
			for old, new := range redactedMap {
				spec = strings.ReplaceAll(spec, old, new)
				resolved = strings.ReplaceAll(resolved, old, new)
			}
		}

		cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", resolved, stateStr))
		if spec != resolved {
			cacheLines = append(cacheLines, fmt.Sprintf("block_exists:%s=%s", spec, stateStr))
		}
	}

	var outFstab []byte
	if unredacted {
		outFstab = []byte(fstabStr)
	} else {
		var outStr string
		for _, e := range entries {
			if strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") || len(e.Fields) < 2 {
				// Sanitize comments and short lines as well if they contain mapped sensitive data
				line := e.RawLine
				if !unredacted {
				    // Since comments could have completely unknown secrets, if it's not unredacted we shouldn't leak them.
				    // However replacing the whole comment with a placeholder is better.
				    if strings.HasPrefix(strings.TrimSpace(e.RawLine), "#") {
				        line = "# <sanitized-comment>"
				    } else if len(e.Fields) == 1 {
				        line = "<sanitized-malformed-line>"
				    }
				}
				outStr += line + "\n"
				continue
			}

			// Extract field 0 and 1 safely, then break them down if they contain specific syntax (like UUID=)
			spec := e.Fields[0]
			target := e.Fields[1]

			// Only sanitize if they are path-like or explicit ID-like, don't blindly replace whole fields
			// if they weren't mapped from block devices.
			if !unredacted {
				if strings.HasPrefix(spec, "UUID=") {
					val := strings.TrimPrefix(spec, "UUID=")
					newVal := sanitizeStr(val, "uuid", redactedMap)
					redactedMap[spec] = "UUID=" + newVal
				} else if strings.HasPrefix(spec, "LABEL=") {
					val := strings.TrimPrefix(spec, "LABEL=")
					newVal := sanitizeStr(val, "label", redactedMap)
					redactedMap[spec] = "LABEL=" + newVal
				} else if strings.HasPrefix(spec, "PARTUUID=") {
					val := strings.TrimPrefix(spec, "PARTUUID=")
					newVal := sanitizeStr(val, "partuuid", redactedMap)
					redactedMap[spec] = "PARTUUID=" + newVal
				} else if strings.HasPrefix(spec, "/") {
					_ = sanitizeStr(spec, "path", redactedMap)
				} else if strings.Contains(spec, ":/") || strings.HasPrefix(spec, "//") {
				    // Network source
				    _ = sanitizeStr(spec, "net", redactedMap)
				}
				if strings.HasPrefix(target, "/") {
					_ = sanitizeStr(target, "path", redactedMap)
				}
			}

			line := e.RawLine

			// Try to sanitize mount options fields if they contain secrets
			if len(e.Fields) >= 4 {
			    opts := e.Fields[3]
			    for _, opt := range strings.Split(opts, ",") {
			        if strings.HasPrefix(opt, "password=") || strings.HasPrefix(opt, "username=") || strings.HasPrefix(opt, "credentials=") || strings.HasPrefix(opt, "secret=") {
			            parts := strings.SplitN(opt, "=", 2)
			            if len(parts) == 2 {
			                sanitizedVal := sanitizeStr(parts[1], "secret", redactedMap)
			                redactedMap[opt] = parts[0] + "=" + sanitizedVal
			            }
			        }
			    }
			}

			for _, f := range e.Fields {
				repl, ok := redactedMap[f]
				if !ok {
				    for oldVal, newVal := range redactedMap {
				        if strings.Contains(f, oldVal) {
				            line = strings.Replace(line, oldVal, newVal, 1)
				        }
				    }
				} else {
					line = strings.Replace(line, f, repl, 1)
				}
			}
			outStr += line + "\n"
		}
		outFstab = []byte(outStr)
	}
	ar.Files = append(ar.Files, x_txtar.File{Name: "input/etc/fstab", Data: outFstab})

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

	if len(s.Fstab) == 0 {
	    return nil, fmt.Errorf("invalid archive: missing fstab data")
	}
	return s, nil
}
