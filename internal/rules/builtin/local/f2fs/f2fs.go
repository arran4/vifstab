package f2fs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type F2fsRule struct{}

func init() {
	rules.Register(&F2fsRule{})
}

func (r *F2fsRule) Name() string {
	return "f2fs"
}

func (r *F2fsRule) Dependencies() []string {
	return []string{}
}

func (r *F2fsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "f2fs" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := rules.SplitOptions(entry.Fields[3])

	hasRo := false
	hasNoRecovery := false
	hasDisableRollForward := false

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		if key == "ro" {
			hasRo = true
		} else if key == "rw" {
			hasRo = false
		} else if key == "norecovery" {
			hasNoRecovery = true
		} else if key == "disable_roll_forward" {
			hasDisableRollForward = true
		}
	}

	if hasNoRecovery && !hasRo {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("f2fs mount option 'norecovery' requires 'ro' on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}
	if hasNoRecovery && !hasDisableRollForward {
		// "norecovery" effectively implies "disable_roll_forward". If they provide norecovery,
		// the kernel treats it as disable_roll_forward, ro.
		// However, the docs say: "norecovery: Disable the roll-forward recovery routine, mounted read- only (i.e., -o ro,disable_roll_forward)"
		// So not strictly requiring it if the kernel does it, but maybe just check if it's there? Actually no need.
	}

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		hasVal := len(parts) > 1
		validator, ok := f2fsOptions[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown f2fs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, "f2fs", key, val, hasVal)...)
		}

		switch key {
		case "heap", "no_heap":
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("f2fs mount option '%s' is deprecated on line %d", key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		case "active_logs":
			if val != "" {
				if val != "2" && val != "4" && val != "6" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for active_logs on line %d, must be 2, 4, or 6", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "mode":
			if val != "" && val != "adaptive" && val != "lfs" && val != "fragment:segment" && val != "fragment:block" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for mode on line %d, must be 'adaptive', 'lfs', 'fragment:segment', or 'fragment:block'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "alloc_mode":
			if val != "" && val != "reuse" && val != "default" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for alloc_mode on line %d, must be 'reuse' or 'default'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "fsync_mode":
			if val != "" && val != "posix" && val != "strict" && val != "nobarrier" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fsync_mode on line %d, must be 'posix', 'strict', or 'nobarrier'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "test_dummy_encryption":
			if val != "" && val != "v1" && val != "v2" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for test_dummy_encryption on line %d, must be 'v1' or 'v2'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "checkpoint":
			if val != "" {
				cpParts := strings.Split(val, ":")
				if cpParts[0] != "enable" && cpParts[0] != "disable" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for checkpoint on line %d, must start with 'enable' or 'disable'", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "compress_algorithm":
			if val != "" {
				algoParts := strings.Split(val, ":")
				algo := algoParts[0]
				if algo != "lzo" && algo != "lz4" && algo != "zstd" && algo != "lzo-rle" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for compress_algorithm on line %d, must be 'lzo', 'lz4', 'zstd', or 'lzo-rle'", algo, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else if len(algoParts) > 1 {
					level, err := strconv.Atoi(algoParts[1])
					if err != nil {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid level '%s' for compress_algorithm on line %d, must be an integer", algoParts[1], entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					} else {
						if algo == "lz4" && (level < 3 || level > 16) {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Invalid level %d for lz4 on line %d, must be between 3 and 16", level, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						} else if algo == "zstd" && (level < 1 || level > 22) {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Invalid level %d for zstd on line %d, must be between 1 and 22", level, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						}
					}
				}
			}
		case "compress_mode":
			if val != "" && val != "fs" && val != "user" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for compress_mode on line %d, must be 'fs' or 'user'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "discard_unit":
			if val != "" && val != "block" && val != "segment" && val != "section" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for discard_unit on line %d, must be 'block', 'segment', or 'section'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "memory":
			if val != "" && val != "normal" && val != "low" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for memory on line %d, must be 'normal' or 'low'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "" && val != "panic" && val != "continue" && val != "remount-ro" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for errors on line %d, must be 'panic', 'continue', or 'remount-ro'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "lookup_mode":
			if val != "" && val != "perf" && val != "compat" && val != "auto" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for lookup_mode on line %d, must be 'perf', 'compat', or 'auto'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "jqfmt":
			if val != "" && val != "vfsold" && val != "vfsv0" && val != "vfsv1" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for jqfmt on line %d, must be 'vfsold', 'vfsv0', or 'vfsv1'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var f2fsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"background_gc":         rules.OptionalStringOption{},
	"gc_merge":              rules.NoValueOption{},
	"nogc_merge":            rules.NoValueOption{},
	"disable_roll_forward":  rules.NoValueOption{},
	"norecovery":            rules.NoValueOption{},
	"discard":               rules.NoValueOption{},
	"nodiscard":             rules.NoValueOption{},
	"heap":                  rules.NoValueOption{},
	"no_heap":               rules.NoValueOption{},
	"nouser_xattr":          rules.NoValueOption{},
	"noacl":                 rules.NoValueOption{},
	"active_logs":           rules.StringOption{},
	"disable_ext_identify":  rules.NoValueOption{},
	"inline_xattr":          rules.NoValueOption{},
	"noinline_xattr":        rules.NoValueOption{},
	"inline_xattr_size":     rules.StringOption{},
	"inline_data":           rules.NoValueOption{},
	"inline_dentry":         rules.NoValueOption{},
	"noinline_dentry":       rules.NoValueOption{},
	"flush_merge":           rules.NoValueOption{},
	"nobarrier":             rules.NoValueOption{},
	"barrier":               rules.NoValueOption{},
	"fastboot":              rules.NoValueOption{},
	"extent_cache":          rules.NoValueOption{},
	"noextent_cache":        rules.NoValueOption{},
	"noinline_data":         rules.NoValueOption{},
	"data_flush":            rules.NoValueOption{},
	"reserve_root":          rules.StringOption{},
	"reserve_node":          rules.StringOption{},
	"resuid":                rules.UIDOption{},
	"resgid":                rules.GIDOption{},
	"fault_injection":       rules.StringOption{},
	"fault_type":            rules.StringOption{},
	"mode":                  rules.StringOption{},
	"usrquota":              rules.NoValueOption{},
	"grpquota":              rules.NoValueOption{},
	"prjquota":              rules.NoValueOption{},
	"usrjquota":             rules.OptionalStringOption{},
	"grpjquota":             rules.OptionalStringOption{},
	"prjjquota":             rules.OptionalStringOption{},
	"jqfmt":                 rules.StringOption{},
	"quota":                 rules.NoValueOption{},
	"noquota":               rules.NoValueOption{},
	"alloc_mode":            rules.StringOption{},
	"fsync_mode":            rules.StringOption{},
	"test_dummy_encryption": rules.OptionalStringOption{},
	"checkpoint":            rules.StringOption{},
	"checkpoint_merge":      rules.NoValueOption{},
	"nocheckpoint_merge":    rules.NoValueOption{},
	"compress_algorithm":    rules.StringOption{},
	"compress_log_size":     rules.StringOption{},
	"compress_extension":    rules.StringOption{},
	"nocompress_extension":  rules.StringOption{},
	"compress_chksum":       rules.NoValueOption{},
	"compress_mode":         rules.StringOption{},
	"compress_cache":        rules.NoValueOption{},
	"inlinecrypt":           rules.NoValueOption{},
	"atgc":                  rules.NoValueOption{},
	"discard_unit":          rules.StringOption{},
	"memory":                rules.StringOption{},
	"age_extent_cache":      rules.NoValueOption{},
	"errors":                rules.StringOption{},
	"nat_bits":              rules.NoValueOption{},
	"lookup_mode":           rules.StringOption{},
})

