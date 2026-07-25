package rules

import (

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
	val := strings.TrimSpace(matches[2])

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
