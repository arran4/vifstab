package fat

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "fat", fatOptions), nil
}

var fatOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"blocksize": rules.EnumOption{
		Allowed:      map[string]bool{"512": true, "1024": true, "2048": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'blocksize' on line %d (only 512, 1024, or 2048 are supported)",
	},
	"uid":         rules.UIDOption{},
	"gid":         rules.GIDOption{},
	"umask":       rules.OctalOption{},
	"dmask":       rules.OctalOption{},
	"fmask":       rules.OctalOption{},
	"allow_utime": rules.OctalOption{},
	"check": rules.EnumOption{
		Allowed:      map[string]bool{"s": true, "strict": true, "r": true, "relaxed": true, "n": true, "normal": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'check' on line %d (only 's', 'r', or 'n' are supported)",
	},
	"codepage":    rules.IntOption{},
	"conv":        rules.StringOption{},
	"cvf_format":  rules.StringOption{},
	"cvf_option":  rules.StringOption{},
	"debug":       rules.NoValueOption{},
	"discard":     rules.NoValueOption{},
	"dos1xfloppy": rules.BoolOption{},
	"errors":      rules.EnumOption{Allowed: map[string]bool{"panic": true, "continue": true, "remount-ro": true}},
	"fat": rules.EnumOption{
		Allowed:      map[string]bool{"12": true, "16": true, "32": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'fat' on line %d (only 12, 16, or 32 are supported)",
	},
	"iocharset": rules.StringOption{},
	"nfs": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"tz": rules.EnumOption{
		Allowed:      map[string]bool{"UTC": true},
		CustomErrMsg: "Invalid value '%s' for fat mount option 'tz' on line %d (only 'UTC' is supported)",
	},
	"time_offset":   rules.IntOption{},
	"quiet":         rules.NoValueOption{},
	"rodir":         rules.NoValueOption{},
	"showexec":      rules.NoValueOption{},
	"sys_immutable": rules.NoValueOption{},
	"flush":         rules.NoValueOption{},
	"usefree":       rules.NoValueOption{},
	"dots":          rules.NoValueOption{},
	"nodots":        rules.NoValueOption{},
	"dotsOK":        rules.BoolOption{},
})
