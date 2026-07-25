package kernel

import (

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

func GetHostKernelVersion(api rules.API) Ver {
	s := api.GetHostKernelVersion()
    if s != "" {
        maj, min, pat := parseKernelVersionStr(s)
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
