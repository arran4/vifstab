package xfs

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/builtin/util"

	"github.com/arran4/vifstab/internal/rules"
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

func splitOptions(s string) []string {
	var opts []string
	var current strings.Builder
	inQuote := false
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == ',' && !inQuote {
			opts = append(opts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		opts = append(opts, current.String())
	}
	return opts
}

func unescapeFstab(s string) string {
	s = strings.ReplaceAll(s, "\\040", " ")
	s = strings.ReplaceAll(s, "\\011", "\t")
	s = strings.ReplaceAll(s, "\\012", "\n")
	s = strings.ReplaceAll(s, "\\134", "\\")
	return s
}

func (r *XfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
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
	options := splitOptions(entry.Fields[3])

	c := rules.CacheFromContext(ctx)
	hostVer := kernel.GetHostKernelVersion(c)

	hasNoRecovery := false
	hasRo := false

	sunit := -1
	swidth := -1

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		if key == "norecovery" {

			hasNoRecovery = true
		} else if key == "ro" {
			hasRo = true
		} else if key == "rw" {
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

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := xfsKnownOptions[key]; !ok {
			// Ignore systemd and userspace extensions
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown XFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
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

		if val != "" {
			if _, ok := xfsValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("XFS mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
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
				// We won't validate exact value for logbsize (16k, 32k, 64k, etc), allocsize (power of 2),
				// or sunit/swidth (512-byte blocks) deeply here to avoid over-complicating without context,
				// but we ensure a value is provided and roughly checks format for logbsize/allocsize ending in k/m/g or digit.
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
				} else { // sunit, swidth should be integers
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
						if key == "sunit" {
							sunit = num
						} else if key == "swidth" {
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
				if util.IsBlockLookup(decodedVal) {
					// Assume valid for LABEL=, UUID=, PARTLABEL=, PARTUUID=, ID=
					continue
				}
				if c != nil {
					if v, ok := c.Get(rules.CacheKey("file_exists:" + decodedVal)); ok {
						if exists, ok := v.(bool); ok && !exists {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Device '%s' specified in XFS mount option %s does not exist on line %d", decodedVal, key, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						}
						continue
					}
				}
				if c == nil {
					if _, err := os.Stat(decodedVal); err != nil {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Device '%s' specified in XFS mount option %s does not exist on line %d", decodedVal, key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
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

var xfsKnownOptions = map[string]struct{}{
	"allocsize": {}, "attr2": {}, "noattr2": {}, "dax": {},
	"discard": {}, "nodiscard": {}, "grpid": {}, "bsdgroups": {},
	"nogrpid": {}, "sysvgroups": {}, "filestreams": {}, "ikeep": {},
	"noikeep": {}, "inode32": {}, "inode64": {}, "largeio": {},
	"nolargeio": {}, "logbufs": {}, "logbsize": {}, "logdev": {},
	"rtdev": {}, "noalign": {}, "norecovery": {}, "nouuid": {},
	"noquota": {}, "uquota": {}, "usrquota": {}, "quota": {},
	"uqnoenforce": {}, "qnoenforce": {}, "gquota": {}, "grpquota": {},
	"gqnoenforce": {}, "pquota": {}, "prjquota": {}, "pqnoenforce": {},
	"sunit": {}, "swidth": {}, "swalloc": {}, "wsync": {},

	// Removed options
	"delaylog": {}, "nodelaylog": {}, "ihashsize": {}, "irixsgid": {},
	"osyncisdsync": {}, "osyncisosync": {}, "barrier": {}, "nobarrier": {},

	// Common generic ones
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},

	// More generic ones
	"_netdev": {}, "dev": {}, "atime": {}, "diratime": {}, "noiversion": {},
	"nolazytime": {}, "owner": {}, "group": {},
}

var xfsValuedOptions = map[string]struct{}{
	"allocsize": {}, "dax": {}, "logbufs": {}, "logbsize": {}, "logdev": {},
	"rtdev": {}, "sunit": {}, "swidth": {}, "context": {}, "defcontext": {},
	"rootcontext": {}, "fscontext": {},
}

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
