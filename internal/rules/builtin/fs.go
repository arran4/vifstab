package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

var knownExternalFS = map[string]bool{
	"zfs":       true,
	"openafs":   true,
	"lustre":    true,
	"beegfs":    true,
	"gpfs":      true,
	"vxfs":      true,
	"9p":        true,
	"ksmbd":     true,
	"gocryptfs": true,
	"rclone":    true,
}

const ProcFilesystemsKey rules.CacheKey = "proc_filesystems"

type FSRule struct{}

func init() {
	rules.Register(&FSRule{})
}

func (r *FSRule) Name() string {
	return "linux_fs"
}

func (r *FSRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *FSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}

	if len(entry.Fields) < 3 {
		return nil, nil // Leave to format rule
	}

	mountPoint := entry.Fields[1]
	fsType := entry.Fields[2]

	if mountPoint == "none" && fsType == "none" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Filesystem type for swap must be 'swap', not 'none' on line %d", entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}, nil
	}

	if fsType == "ignore" || fsType == "swap" || fsType == "none" || fsType == "bind" {
		return nil, nil
	}

	// Known external filesystems not in /proc/filesystems
	if knownExternalFS[fsType] {
		return nil, nil
	}
	// Known external filesystems or FUSE wrappers not in /proc/filesystems
	if strings.HasPrefix(fsType, "fuse.") || fsType == "fuse" || fsType == "fuseblk" || fsType == "fusexmp" {
		return nil, nil
	}

	c := rules.CacheFromContext(ctx)
	if c == nil {
		return nil, fmt.Errorf("no cache provided in context")
	}

	val, ok := c.Get(ProcFilesystemsKey)
	if !ok {
		// Try to load it
		fsMap, err := LoadProcFilesystems("/proc/filesystems")
		if err != nil {
			return nil, fmt.Errorf("failed to load proc filesystems: %w", err)
		}
		c.Set(ProcFilesystemsKey, fsMap)
		val = fsMap
	}

	validFS, ok := val.(map[string]bool)
	if !ok {
		return nil, fmt.Errorf("invalid cache type for proc_filesystems")
	}

	if !validFS[fsType] {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Unknown filesystem type '%s' on line %d", fsType, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}, nil
	}

	return nil, nil
}

func LoadProcFilesystems(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	fsMap := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 {
			fsMap[fields[len(fields)-1]] = true
		}
	}
	return fsMap, scanner.Err()
}
