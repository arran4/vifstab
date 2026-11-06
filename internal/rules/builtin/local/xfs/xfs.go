package xfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type XfsRule struct{}

func init() {
	rules.Register(&XfsRule{})
}

func (r *XfsRule) Name() string {
	return "xfs"
}

func (r *XfsRule) Dependencies() []string {
	return []string{}
}

func unescapeFstab(s string) string {
	s = strings.ReplaceAll(s, "\\040", " ")
	s = strings.ReplaceAll(s, "\\011", "\t")
	s = strings.ReplaceAll(s, "\\012", "\n")
	s = strings.ReplaceAll(s, "\\134", "\\")
	return s
}

func (r *XfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "xfs" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	parsedOptions := rules.SplitOptions(entry.Fields[3])
	hostVer := kernel.GetHostKernelVersion(api)

	hasNoRecovery := false
	hasRo := false

	sunit := -1
	swidth := -1

	for _, opt := range parsedOptions {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		switch key {
		case "norecovery":
			hasNoRecovery = true
		case "ro":
			hasRo = true
		case "rw":
			hasRo = false
		}
	}

	if hasNoRecovery && !hasRo {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("XFS mount option 'norecovery' requires 'ro' on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}

	for _, opt := range parsedOptions {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasEquals := len(parts) > 1
		if hasEquals {
			val = parts[1]
		}

		if extensionIssues, handled := options.ValidateExtensionOption(ctx, api, entry, "XFS", key, val, len(parts) > 1); handled {
			issues = append(issues, extensionIssues...)
			continue
		}

		validator, ok := xfsOptions[key]
		if !ok {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("Unknown XFS mount option '%s' on line %d", key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
			continue
		}

		if rm, ok := xfsRemovedOptions[key]; ok {
			if hostVer.Major == 0 || hostVer.AtLeast(rm.major, rm.minor, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("XFS mount option '%s' was removed in kernel %d.%d and will yield mount failures on line %d", key, rm.major, rm.minor, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		switch key {
		case "attr2", "noattr2", "ikeep", "noikeep":
			if hostVer.Major == 0 || hostVer.AtLeast(5, 10, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("XFS mount option '%s' is deprecated as of kernel v5.10 and will be removed/immutable on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		case "dax":
			if val != "" {
				if hostVer.Major != 0 && !hostVer.AtLeast(5, 8, 0) {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Valued DAX options require kernel 5.8 or newer on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
				if val != "inode" && val != "never" && val != "always" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for dax on line %d, must be 'inode', 'never', or 'always'", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "logbufs":
			if val == "" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for XFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else {
				if num, err := strconv.Atoi(val); err == nil {
					if num < 2 || num > 8 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value '%s' for logbufs on line %d, must be between 2 and 8 inclusive", val, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				} else {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for logbufs on line %d, must be an integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "logbsize", "allocsize", "sunit", "swidth":
			if val == "" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for XFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else {
				if key == "logbsize" || key == "allocsize" {
					numStr := val
					lastChar := val[len(val)-1]
					if lastChar == 'k' || lastChar == 'K' || lastChar == 'm' || lastChar == 'M' || lastChar == 'g' || lastChar == 'G' {
						numStr = val[:len(val)-1]
					}
					if num, err := strconv.Atoi(numStr); err != nil || num < 0 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value format '%s' for %s on line %d", val, key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				} else {
					if num, err := strconv.Atoi(val); err != nil {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value '%s' for %s on line %d, must be an integer", val, key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					} else if num < 0 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value '%s' for %s on line %d, must be non-negative", val, key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					} else {
						switch key {
						case "sunit":
							sunit = num
						case "swidth":
							swidth = num
						}
					}
				}
			}
		case "logdev", "rtdev":
			if val == "" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing device for XFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else {
				decodedVal := unescapeFstab(val)
				if api.CheckBlockDeviceExists(decodedVal) == rules.DoesNotExist {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Device '%s' specified in XFS mount option %s does not exist on line %d", decodedVal, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		default:
			if validator != nil {
				issues = append(issues, validator.ValidateOption(ctx, api, entry, "XFS", key, val, hasEquals, nil)...)
			}
		}
	}

	if sunit != -1 && swidth != -1 && sunit != 0 {
		if swidth%sunit != 0 {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("XFS mount option 'swidth' (%d) must be a multiple of 'sunit' (%d) on line %d", swidth, sunit, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	}

	return issues, nil
}

var xfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"allocsize":   options.StringOption{},
	"attr2":       options.NoValueOption{},
	"noattr2":     options.NoValueOption{},
	"dax":         options.OptionalStringOption{},
	"discard":     options.NoValueOption{},
	"nodiscard":   options.NoValueOption{},
	"grpid":       options.NoValueOption{},
	"bsdgroups":   options.NoValueOption{},
	"nogrpid":     options.NoValueOption{},
	"sysvgroups":  options.NoValueOption{},
	"filestreams": options.NoValueOption{},
	"ikeep":       options.NoValueOption{},
	"noikeep":     options.NoValueOption{},
	"inode32":     options.NoValueOption{},
	"inode64":     options.NoValueOption{},
	"largeio":     options.NoValueOption{},
	"nolargeio":   options.NoValueOption{},
	"logbufs":     options.StringOption{},
	"logbsize":    options.StringOption{},
	"logdev":      options.StringOption{},
	"rtdev":       options.StringOption{},
	"noalign":     options.NoValueOption{},
	"norecovery":  options.NoValueOption{},
	"nouuid":      options.NoValueOption{},
	"noquota":     options.NoValueOption{},
	"uquota":      options.NoValueOption{},
	"usrquota":    options.NoValueOption{},
	"quota":       options.NoValueOption{},
	"uqnoenforce": options.NoValueOption{},
	"qnoenforce":  options.NoValueOption{},
	"gquota":      options.NoValueOption{},
	"grpquota":    options.NoValueOption{},
	"gqnoenforce": options.NoValueOption{},
	"pquota":      options.NoValueOption{},
	"prjquota":    options.NoValueOption{},
	"pqnoenforce": options.NoValueOption{},
	"sunit":       options.StringOption{},
	"swidth":      options.StringOption{},
	"swalloc":     options.NoValueOption{},
	"wsync":       options.NoValueOption{},

	// Removed options
	"delaylog":     options.NoValueOption{},
	"nodelaylog":   options.NoValueOption{},
	"ihashsize":    options.NoValueOption{},
	"irixsgid":     options.NoValueOption{},
	"osyncisdsync": options.NoValueOption{},
	"osyncisosync": options.NoValueOption{},
	"barrier":      options.NoValueOption{},
	"nobarrier":    options.NoValueOption{},
})

var xfsRemovedOptions = map[string]struct {
	major int
	minor int
}{
	"delaylog": {4, 0}, "nodelaylog": {4, 0},
	"ihashsize":    {4, 0},
	"irixsgid":     {4, 0},
	"osyncisdsync": {4, 0}, "osyncisosync": {4, 0},
	"barrier": {4, 19}, "nobarrier": {4, 19},
}
