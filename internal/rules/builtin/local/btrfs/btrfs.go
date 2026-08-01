package btrfs

import (
	"strconv"

	"context"
	"os"
	"strings"

	"fmt"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type BTRFSRule struct{}

func init() {
	rules.Register(&BTRFSRule{})
}

func (r *BTRFSRule) Name() string {
	return "btrfs"
}

func (r *BTRFSRule) Dependencies() []string {
	return []string{}
}

func (r *BTRFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "btrfs" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := strings.Split(entry.Fields[3], ",")

	c := rules.CacheFromContext(ctx)

	hostVer := kernel.GetHostKernelVersion(api)

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := btrfsOptions[key]; !ok {
			// Ignore systemd extensions
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown BTRFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		switch key {
		case "autodefrag", "noautodefrag", "inode_cache", "noinode_cache", "check_int", "check_int_data", "check_int_print_mask":
			if !hostVer.AtLeast(3, 0, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option '%s' requires kernel 3.0 or newer on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			if (key == "inode_cache" || key == "noinode_cache") && hostVer.AtLeast(5, 11, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option '%s' was removed in kernel 5.11 on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if (key == "check_int" || key == "check_int_data" || key == "check_int_print_mask") && hostVer.AtLeast(6, 7, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option '%s' was removed in kernel 6.7 on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "commit", "rescan_uuid_tree":
			if !hostVer.AtLeast(3, 12, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option '%s' requires kernel 3.12 or newer on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			if key == "commit" && val != "" {
				if v, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for commit on line %d, must be an integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else if v > 300 {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Value '%d' for commit is larger than recommended 300s on line %d", v, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelWarning,
					})
				}
			}
		case "fatal_errors":
			if !hostVer.AtLeast(3, 4, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'fatal_errors' requires kernel 3.4 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			if val != "bug" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fatal_errors on line %d, must be 'bug' or 'panic'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "skip_balance":
			if !hostVer.AtLeast(3, 3, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'skip_balance' requires kernel 3.3 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "usebackuproot":
			if !hostVer.AtLeast(4, 6, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'usebackuproot' requires kernel 4.6 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "recovery":
			if !hostVer.AtLeast(3, 2, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'recovery' requires kernel 3.2 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			if hostVer.AtLeast(4, 5, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'recovery' is deprecated since kernel 4.5 on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "norecovery":
			if !hostVer.AtLeast(4, 5, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'norecovery' requires kernel 4.5 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "nospace_cache":
			if !hostVer.AtLeast(3, 2, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'nospace_cache' requires kernel 3.2 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "max_inline":
			if val != "" {
				if !isValidMaxInline(val) {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for max_inline on line %d, must be a byte size (e.g. 2048, 2K)", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "thread_pool", "subvolid", "metadata_ratio":
			if val != "" {
				if _, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for %s on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "fragment":
			if !hostVer.AtLeast(4, 4, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'fragment' requires kernel 4.4 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			if val != "data" && val != "metadata" && val != "all" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fragment on line %d, must be 'data', 'metadata' or 'all'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "space_cache":
			if val != "" && val != "v1" && val != "v2" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for space_cache on line %d, must be 'v1' or 'v2'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if (val == "v1" || val == "v2") && !hostVer.AtLeast(4, 5, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'space_cache=%s' requires kernel 4.5 or newer on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "rescue":
			if !hostVer.AtLeast(5, 9, 0) && hostVer.Major != 0 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("BTRFS mount option 'rescue' requires kernel 5.9 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
			rescueOpts := strings.Split(val, ":")
			for _, ropt := range rescueOpts {
				if ropt == "ignorebadroots" || ropt == "ibadroots" || ropt == "ignoredatacsums" || ropt == "idatacsums" {
					if !hostVer.AtLeast(5, 11, 0) && hostVer.Major != 0 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("BTRFS rescue option '%s' requires kernel 5.11 or newer on line %d", ropt, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelWarning,
						})
					}
				} else if ropt == "ignoremetacsums" || ropt == "imetacsums" {
					if !hostVer.AtLeast(6, 12, 0) && hostVer.Major != 0 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("BTRFS rescue option '%s' requires kernel 6.12 or newer on line %d", ropt, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelWarning,
						})
					}
				} else if ropt != "usebackuproot" && ropt != "nologreplay" && ropt != "all" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' in rescue on line %d", ropt, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "compress", "compress-force":
			cparts := strings.SplitN(val, ":", 2)
			algo := cparts[0]
			levelStr := ""
			if len(cparts) > 1 {
				levelStr = cparts[1]
			}
			if algo == "zstd" {
				if !hostVer.AtLeast(4, 14, 0) && hostVer.Major != 0 {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("BTRFS mount option compress algorithm 'zstd' requires kernel 4.14 or newer on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelWarning,
					})
				}
				if levelStr != "" {
					if !hostVer.AtLeast(5, 1, 0) && hostVer.Major != 0 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("BTRFS mount option compress algorithm 'zstd' levels requires kernel 5.1 or newer on line %d", entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelWarning,
						})
					}
					level, err := strconv.Atoi(levelStr)
					if err != nil {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid level '%s' for zstd compress on line %d, must be an integer", levelStr, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					} else {
						if level < -15 || level > 15 {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Invalid level '%d' for zstd compress on line %d, must be between -15 and 15", level, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						}
						if level < 0 && !hostVer.AtLeast(6, 15, 0) && hostVer.Major != 0 {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("BTRFS mount option compress algorithm 'zstd' negative levels requires kernel 6.15 or newer on line %d", entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelWarning,
							})
						}
					}
				}
			}
			if algo == "zlib" {
				if levelStr != "" {
					level, err := strconv.Atoi(levelStr)
					if err != nil {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid level '%s' for zlib compress on line %d, must be an integer", levelStr, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					} else if level < 1 || level > 9 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid level '%d' for zlib compress on line %d, must be between 1 and 9", level, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
			}
			if algo == "lzo" {
				if levelStr != "" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("BTRFS mount option compress algorithm 'lzo' does not support levels on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}

			if algo != "zlib" && algo != "lzo" && algo != "zstd" && algo != "no" && algo != "" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for compress on line %d", algo, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "discard":
			if val == "async" {
				if !hostVer.AtLeast(5, 6, 0) && hostVer.Major != 0 {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("BTRFS mount option 'discard=async' requires kernel 5.6 or newer on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelWarning,
					})
				}
			} else if val != "" && val != "sync" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for discard on line %d, must be 'sync' or 'async'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "device":
			if val != "" {
				if !api.CheckBlockDeviceExists(val) {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Device '%s' specified in BTRFS mount option device does not exist on line %d", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "subvol":
			if val != "" {
				checked := false
				if c != nil {
					if v, ok := c.Get(rules.CacheKey("btrfs_subvol_exists:" + val)); ok {
						if exists, ok := v.(bool); ok && !exists {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Subvolume '%s' might not exist on line %d", val, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelWarning,
							})
						}
						checked = true
					}
				}
				if !checked {
					// We only check if it's an absolute path, because otherwise it's relative
					// to the BTRFS subvolume root which we can't reliably resolve without parsing UUIDs/mounts.
					if strings.HasPrefix(val, "/") {
						if _, err := os.Stat(val); err != nil && os.IsNotExist(err) {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Subvolume '%s' might not exist on line %d", val, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelWarning,
							})
						}
					}
				}
			}
		}
	}

	return issues, nil
}

var btrfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":                    options.NoValueOption{},
	"noacl":                  options.NoValueOption{},
	"autodefrag":             options.NoValueOption{},
	"noautodefrag":           options.NoValueOption{},
	"barrier":                options.NoValueOption{},
	"nobarrier":              options.NoValueOption{},
	"clear_cache":            options.NoValueOption{},
	"commit":                 options.StringOption{},
	"compress":               options.OptionalStringOption{},
	"compress-force":         options.OptionalStringOption{},
	"datacow":                options.NoValueOption{},
	"nodatacow":              options.NoValueOption{},
	"datasum":                options.NoValueOption{},
	"nodatasum":              options.NoValueOption{},
	"degraded":               options.NoValueOption{},
	"device":                 options.StringOption{},
	"discard":                options.OptionalStringOption{},
	"nodiscard":              options.NoValueOption{},
	"enospc_debug":           options.NoValueOption{},
	"noenospc_debug":         options.NoValueOption{},
	"fatal_errors":           options.StringOption{},
	"flushoncommit":          options.NoValueOption{},
	"noflushoncommit":        options.NoValueOption{},
	"fragment":               options.StringOption{},
	"nologreplay":            options.NoValueOption{},
	"max_inline":             options.StringOption{},
	"metadata_ratio":         options.StringOption{},
	"norecovery":             options.NoValueOption{},
	"rescan_uuid_tree":       options.NoValueOption{},
	"rescue":                 options.OptionalStringOption{},
	"skip_balance":           options.NoValueOption{},
	"space_cache":            options.OptionalStringOption{},
	"nospace_cache":          options.NoValueOption{},
	"ssd":                    options.NoValueOption{},
	"ssd_spread":             options.NoValueOption{},
	"nossd":                  options.NoValueOption{},
	"nossd_spread":           options.NoValueOption{},
	"subvol":                 options.StringOption{},
	"subvolid":               options.StringOption{},
	"thread_pool":            options.StringOption{},
	"treelog":                options.NoValueOption{},
	"notreelog":              options.NoValueOption{},
	"usebackuproot":          options.NoValueOption{},
	"user_subvol_rm_allowed": options.NoValueOption{},
	"recovery":               options.NoValueOption{},
	"inode_cache":            options.NoValueOption{},
	"noinode_cache":          options.NoValueOption{},
	"check_int":              options.OptionalStringOption{},
	"check_int_data":         options.OptionalStringOption{},
	"check_int_print_mask":   options.OptionalStringOption{},
})

func isValidMaxInline(val string) bool {
	if val == "" {
		return false
	}

	valUpper := strings.ToUpper(val)
	lastChar := valUpper[len(valUpper)-1]

	numPart := valUpper
	if lastChar == 'K' || lastChar == 'M' || lastChar == 'G' || lastChar == 'T' || lastChar == 'P' {
		numPart = valUpper[:len(valUpper)-1]
	}

	if numPart == "" {
		return false
	}

	for _, ch := range numPart {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}
