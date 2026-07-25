package util

import (
	"regexp"
	"strings"
)

var blockLookupRe = regexp.MustCompile(`^(LABEL|UUID|PARTLABEL|PARTUUID|ID)=.*`)

func ResolveDevicePath(dev string) string {
	if !blockLookupRe.MatchString(dev) {
		return dev
	}
	parts := strings.SplitN(dev, "=", 2)
	if len(parts) != 2 {
		return dev
	}
	key := parts[0]
	val := parts[1]

	// Unescape if quoted
	if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
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
