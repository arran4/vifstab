package kernel

import (
	"os"
	"regexp"
	"strconv"

	"github.com/arran4/vifstab/internal/rules"
)

var kernelVersionRe = regexp.MustCompile(`(?:Linux version )?(\d+)\.(\d+)(?:\.(\d+))?`)

func parseKernelVersionStr(v string) (major, minor, patch int) {
	matches := kernelVersionRe.FindStringSubmatch(v)
	if len(matches) > 0 {
		major, _ = strconv.Atoi(matches[1])
		minor, _ = strconv.Atoi(matches[2])
		if len(matches) > 3 && matches[3] != "" {
			patch, _ = strconv.Atoi(matches[3])
		}
	}
	return
}

func GetHostKernelVersion(c *rules.Cache) Ver {
	if c != nil {
		if v, ok := c.Get("proc_version"); ok {
			if b, ok := v.([]byte); ok {
				maj, min, pat := parseKernelVersionStr(string(b))
				return Ver{Major: maj, Minor: min, Patch: pat}
			} else if s, ok := v.(string); ok {
				maj, min, pat := parseKernelVersionStr(s)
				return Ver{Major: maj, Minor: min, Patch: pat}
			}
		}
		if v, ok := c.Get("proc_osrelease"); ok {
			if b, ok := v.([]byte); ok {
				maj, min, pat := parseKernelVersionStr(string(b))
				return Ver{Major: maj, Minor: min, Patch: pat}
			} else if s, ok := v.(string); ok {
				maj, min, pat := parseKernelVersionStr(s)
				return Ver{Major: maj, Minor: min, Patch: pat}
			}
		}
	}

	b, err := os.ReadFile("/proc/version")
	if err == nil {
		maj, min, pat := parseKernelVersionStr(string(b))
		return Ver{Major: maj, Minor: min, Patch: pat}
	}
	b, err = os.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil {
		maj, min, pat := parseKernelVersionStr(string(b))
		return Ver{Major: maj, Minor: min, Patch: pat}
	}
	return Ver{Major: 0, Minor: 0, Patch: 0}
}

type Ver struct {
	Major int
	Minor int
	Patch int
}

func (v Ver) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

func (v Ver) AtLeast(major, minor, patch int) bool {
	if v.IsZero() {
		return false // if version unknown, we can't definitively say it's at least
	}
	if v.Major > major {
		return true
	}
	if v.Major == major {
		if v.Minor > minor {
			return true
		}
		if v.Minor == minor {
			return v.Patch >= patch
		}
	}
	return false
}

func (v Ver) LessThan(major, minor, patch int) bool {
	if v.IsZero() {
		return false // if version unknown, we can't definitively say it's less than
	}
	if v.Major < major {
		return true
	}
	if v.Major == major {
		if v.Minor < minor {
			return true
		}
		if v.Minor == minor {
			return v.Patch < patch
		}
	}
	return false
}

func (v Ver) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}
