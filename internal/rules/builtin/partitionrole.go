package builtin

import (
	"context"
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

	isSpecial := fstype == "swap" || fstype == "none" || fstype == "bind" || fstype == "ignore"
	if isSpecial || !isLocalBlockReference(dev) {
		return nil, nil
	}

	inv, err := api.GetBlockDevices()
	if err != nil {
		return nil, nil
	}

	matched := inv.Resolve(dev)

	var issues []rules.ValidationIssue

	if matched == nil {
		isRelevantToPartitionRole := mountpoint == "/boot" || mountpoint == "/efi" || mountpoint == "/boot/efi" || fstype == "vfat" || fstype == "fat"
		if isRelevantToPartitionRole {
			if !inv.Available {
				issues = append(issues, rules.ValidationIssue{
					Message: "Live metadata check unavailable for this block device",
					Line:    entry.RawLine,
					Level:   rules.IssueLevelInfo,
				})
			} else {
				issues = append(issues, rules.ValidationIssue{
					Message: "Source could not be resolved against current live block metadata and partition-role check was skipped",
					Line:    entry.RawLine,
					Level:   rules.IssueLevelInfo,
				})
			}
		}
		return issues, nil
	}

	isESPType := strings.EqualFold(matched.PartType, "c12a7328-f81f-11d2-ba4b-00a0c93ec93b")

	declaredFSType := fstype
	detectedFSType := matched.FSType

	// Determine if the actual filesystem is FAT (prefer live metadata)
	actualIsFAT := false
	if detectedFSType != "" {
		actualIsFAT = (detectedFSType == "vfat" || detectedFSType == "fat" || detectedFSType == "msdos")
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
		for _, d := range inv.Devices {
			// Skip the same canonical device
			if d.DeviceID != "" && matched.DeviceID != "" && d.DeviceID == matched.DeviceID {
				continue
			}
			if d.Path == matched.Path {
				continue
			}
			for _, mp := range d.MountPoints {
				if mp == "/efi" || mp == "/boot/efi" {
					if d.FSType == "vfat" || d.FSType == "fat" || d.FSType == "msdos" {
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

func isLocalBlockReference(source string) bool {
	if strings.HasPrefix(source, "UUID=") ||
		strings.HasPrefix(source, "LABEL=") ||
		strings.HasPrefix(source, "PARTUUID=") ||
		strings.HasPrefix(source, "PARTLABEL=") ||
		strings.HasPrefix(source, "ID=") {
		return true
	}
	if strings.HasPrefix(source, "/dev/disk/by-") {
		return true
	}
	if strings.HasPrefix(source, "/dev/") &&
		!strings.HasPrefix(source, "/dev/mapper/") &&
		!strings.HasPrefix(source, "/dev/loop") &&
		!strings.HasPrefix(source, "/dev/ram") &&
		!strings.HasPrefix(source, "/dev/sr") &&
		!strings.HasPrefix(source, "/dev/fd") {
		return true
	}
	return false
}
