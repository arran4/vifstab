package rules

import (
	"context"
	"os"
	"regexp"
	"strings"
)

var blockLookupRe = regexp.MustCompile(`^(?i)\s*(LABEL|UUID|PARTLABEL|PARTUUID|ID)\s*=\s*(.*)`)

// ResolveDevicePath translates block lookups (like LABEL=, UUID=, etc.) into
// their corresponding /dev/disk/by-* paths. It leaves other paths unchanged.
func ResolveDevicePath(dev string) string {
	matches := blockLookupRe.FindStringSubmatch(dev)
	if len(matches) != 3 {
		return dev
	}
	key := strings.ToUpper(matches[1])
	val := matches[2]

	// Unescape if quoted
	if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
		val = val[1 : len(val)-1]
	} else if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
        val = val[1 : len(val)-1]
    }

	switch key {
	case "LABEL":
		return "/dev/disk/by-label/" + val
	case "UUID":
		return "/dev/disk/by-uuid/" + val
	case "PARTLABEL":
		return "/dev/disk/by-partlabel/" + val
	case "PARTUUID":
		return "/dev/disk/by-partuuid/" + val
	case "ID":
		return "/dev/disk/by-id/" + val
	}
	return dev
}

// CheckFileExists checks if a file exists, taking into account the cache
// and resolving block lookups transparently.
func CheckFileExists(ctx context.Context, path string) bool {
	resolvedPath := ResolveDevicePath(path)
	c := CacheFromContext(ctx)
	if c != nil {
		if v, ok := c.Get(CacheKey("file_exists:" + resolvedPath)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
		// Also check if they cached the raw path for some reason
		if v, ok := c.Get(CacheKey("file_exists:" + path)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}


	}

	_, err := os.Stat(resolvedPath)
	return err == nil
}
