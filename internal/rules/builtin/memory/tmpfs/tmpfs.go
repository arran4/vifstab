package tmpfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type TmpfsRule struct{}

func init() {
	rules.Register(&TmpfsRule{})
}

func (r *TmpfsRule) Name() string {
	return "tmpfs"
}

func (r *TmpfsRule) Dependencies() []string {
	return []string{}
}

func (r *TmpfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "tmpfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "tmpfs", tmpfsOptions), nil
}

var tmpfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"size":                     options.StringOption{},
	"nr_blocks":                options.StringOption{},
	"nr_inodes":                options.StringOption{},
	"noswap":                   options.NoValueOption{},
	"mode":                     options.OctalOption{},
	"gid":                      options.GIDOption{},
	"uid":                      options.UIDOption{},
	"quota":                    options.NoValueOption{},
	"usrquota":                 options.NoValueOption{},
	"grpquota":                 options.NoValueOption{},
	"usrquota_block_hardlimit": options.StringOption{},
	"usrquota_inode_hardlimit": options.StringOption{},
	"grpquota_block_hardlimit": options.StringOption{},
	"grpquota_inode_hardlimit": options.StringOption{},
	"inode32":                  options.NoValueOption{},
	"inode64":                  options.NoValueOption{},
	"casefold":                 options.OptionalStringOption{},
	"strict_encoding":          options.NoValueOption{},
	"huge": options.EnumOption{
		Allowed: map[string]bool{"never": true, "always": true, "within_size": true, "advise": true},
	},
	"mpol": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}

		// Remove quotes if present
		cleanVal := val
		if len(cleanVal) >= 2 && cleanVal[0] == '"' && cleanVal[len(cleanVal)-1] == '"' {
			cleanVal = cleanVal[1 : len(cleanVal)-1]
		}

		// Remove optional flag prefixes before checking prefix logic
		isFlagMatch := false
		flagTrimmed := cleanVal
		for _, prefix := range []string{"default=", "prefer=", "bind=", "interleave=", "local="} {
			if strings.HasPrefix(cleanVal, prefix) {
				remainder := strings.TrimPrefix(cleanVal, prefix)
				if strings.HasPrefix(remainder, "static:") {
					flagTrimmed = prefix[:len(prefix)-1] + ":" + strings.TrimPrefix(remainder, "static:")
					isFlagMatch = true
					break
				} else if strings.HasPrefix(remainder, "relative:") {
					flagTrimmed = prefix[:len(prefix)-1] + ":" + strings.TrimPrefix(remainder, "relative:")
					isFlagMatch = true
					break
				} else if remainder == "static" || remainder == "relative" {
					flagTrimmed = prefix[:len(prefix)-1]
					isFlagMatch = true
					break
				}
			}
		}

		checkVal := cleanVal
		if isFlagMatch {
			checkVal = flagTrimmed
		}

		if !strings.HasPrefix(checkVal, "default") && !strings.HasPrefix(checkVal, "prefer:") && !strings.HasPrefix(checkVal, "bind:") && !strings.HasPrefix(checkVal, "interleave") && !strings.HasPrefix(checkVal, "local") {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for tmpfs mount option 'mpol' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
})
