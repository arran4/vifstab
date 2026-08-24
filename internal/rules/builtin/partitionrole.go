package builtin

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type PartitionRoleRule struct{}

func init() {
	rules.Register(&PartitionRoleRule{})
}

func (r *PartitionRoleRule) Name() string {
	return "PartitionRole"
}

func (r *PartitionRoleRule) Dependencies() []string {
	return nil
}

func (r *PartitionRoleRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) < 3 {
		return nil, nil
	}

	dev := entry.Fields[0]
	mountpoint := entry.Fields[1]
	fstype := entry.Fields[2]

	devs, err := api.GetBlockDevices()
	if err != nil {
		return nil, nil
	}

	matched := findDeviceByPathOrAttr(devs, dev)
	if matched == nil {
		return nil, nil
	}

	var issues []rules.ValidationIssue

	isESPType := strings.EqualFold(matched.PartType, "c12a7328-f81f-11d2-ba4b-00a0c93ec93b")

	declaredFSType := fstype
	detectedFSType := matched.FSType

	// Determine if the actual filesystem is FAT (prefer live metadata)
	actualIsFAT := false
	if detectedFSType != "" {
		actualIsFAT = (detectedFSType == "vfat" || detectedFSType == "fat")
	} else if declaredFSType != "" && declaredFSType != "auto" {
		actualIsFAT = (declaredFSType == "vfat" || declaredFSType == "fat")
	}

	// GPT ESP type on a non-FAT filesystem
	if isESPType {
		if detectedFSType != "" && !actualIsFAT {
			issues = append(issues, rules.ValidationIssue{
				Message:  "GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
				Line:     entry.RawLine,
				Level:    rules.IssueLevelError,
				Severity: 0,
			})
		} else if detectedFSType == "" && declaredFSType != "auto" && declaredFSType != "" && !actualIsFAT {
			issues = append(issues, rules.ValidationIssue{
				Message:  "GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
				Line:     entry.RawLine,
				Level:    rules.IssueLevelError,
				Severity: 0,
			})
		}
	}

	// Warn if concrete declared FS disagrees with concrete detected FS
	if declaredFSType != "" && declaredFSType != "auto" && detectedFSType != "" {
		if declaredFSType != detectedFSType {
			issues = append(issues, rules.ValidationIssue{
				Message: "Declared filesystem type does not match detected live filesystem type",
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		}
	}

	// FAT mounted at `/efi` or `/boot/efi` but not typed as an ESP
	if actualIsFAT && !isESPType && (mountpoint == "/efi" || mountpoint == "/boot/efi") {
		// Only warn if we actually resolved the PartType
		if matched.PartType != "" {
			issues = append(issues, rules.ValidationIssue{
				Message: "FAT filesystem mounted at EFI mountpoint but GPT partition type is not ESP",
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		}
	}

	// `/boot` typed as ESP while a distinct `/efi` FAT partition exists and is the apparent EFI partition
	if isESPType && mountpoint == "/boot" {
		hasEfiFat := false
		for _, d := range devs {
			// Skip the same device seen through another mountpoint or alias
			if d.Path == matched.Path {
				continue
			}
			for _, mp := range d.MountPoints {
				if mp == "/efi" || mp == "/boot/efi" {
					if d.FSType == "vfat" || d.FSType == "fat" {
						hasEfiFat = true
						break
					}
				}
			}
		}
		if hasEfiFat {
			issues = append(issues, rules.ValidationIssue{
				Message: "/boot is typed as ESP but a distinct FAT EFI partition is present",
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		}
	}

	return issues, nil
}

func findDeviceByPathOrAttr(devs []rules.BlockDevice, dev string) *rules.BlockDevice {
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
