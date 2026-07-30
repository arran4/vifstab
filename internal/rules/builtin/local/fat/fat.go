package fat

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type FatRule struct{}

func init() {
	rules.Register(&FatRule{})
}

func (r *FatRule) Name() string {
	return "fat"
}

func (r *FatRule) Dependencies() []string {
	return []string{}
}

func (r *FatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fat" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "fat", fatOptions), nil
}

var fatOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"blocksize": options.EnumOption{
		Allowed:      map[string]bool{"512": true, "1024": true, "2048": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'blocksize' on line %d (only 512, 1024, or 2048 are supported)",
	},
	"uid":         options.UIDOption{},
	"gid":         options.GIDOption{},
	"umask":       options.OctalOption{},
	"dmask":       options.OctalOption{},
	"fmask":       options.OctalOption{},
	"allow_utime": options.OctalOption{},
	"check": options.EnumOption{
		Allowed:      map[string]bool{"s": true, "strict": true, "r": true, "relaxed": true, "n": true, "normal": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'check' on line %d (only 's', 'r', or 'n' are supported)",
	},
	"codepage":    options.IntOption{},
	"conv":        options.StringOption{},
	"cvf_format":  options.StringOption{},
	"cvf_option":  options.StringOption{},
	"debug":       options.NoValueOption{},
	"discard":     options.NoValueOption{},
	"dos1xfloppy": options.BoolOption{},
	"errors":      options.EnumOption{Allowed: map[string]bool{"panic": true, "continue": true, "remount-ro": true}},
	"fat": options.EnumOption{
		Allowed:      map[string]bool{"12": true, "16": true, "32": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'fat' on line %d (only 12, 16, or 32 are supported)",
	},
	"iocharset": options.StringOption{},
	"nfs": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if hasVal && val != "" {
			if val != "stale_rw" && val != "nostale_ro" {
				return []rules.ValidationIssue{
					{
						Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'nfs' on line %d", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					},
				}
			}
		}
		return nil
	}),
	"tz": options.EnumOption{
		Allowed:      map[string]bool{"UTC": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'tz' on line %d (only 'UTC' is supported)",
	},
	"time_offset":   options.IntOption{},
	"quiet":         options.NoValueOption{},
	"rodir":         options.NoValueOption{},
	"showexec":      options.NoValueOption{},
	"sys_immutable": options.NoValueOption{},
	"flush":         options.NoValueOption{},
	"usefree":       options.NoValueOption{},
	"dots":          options.NoValueOption{},
	"nodots":        options.NoValueOption{},
	"dotsOK":        options.BoolOption{},
})
