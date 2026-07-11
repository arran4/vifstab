package builtin

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"

	"fmt"

	"github.com/arran4/vifstab/internal/rules"
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

func (r *BTRFSRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.Suggestion, error) {
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

	var suggestions []rules.Suggestion
	options := strings.Split(entry.Fields[3], ",")

	c := rules.CacheFromContext(ctx)

	maj, min, patch := getHostKernelVersion(c)
	hostVer := kernelVer{major: maj, minor: min, patch: patch}

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := btrfsKnownOptions[key]; !ok {
			// Ignore systemd extensions
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Unknown BTRFS mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
			continue
		}

		switch key {
		case "fatal_errors":
			if val != "bug" && val != "panic" {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Invalid value '%s' for fatal_errors on line %d, must be 'bug' or 'panic'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
		case "fragment":
			if !hostVer.atLeast(4, 4, 0) && hostVer.major != 0 {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("BTRFS mount option 'fragment' requires kernel 4.4 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelWarning,
				})
			}
			if val != "data" && val != "metadata" && val != "all" {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Invalid value '%s' for fragment on line %d, must be 'data', 'metadata' or 'all'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
		case "space_cache":
			if val != "" && val != "v1" && val != "v2" {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Invalid value '%s' for space_cache on line %d, must be 'v1' or 'v2'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
		case "rescue":
			if !hostVer.atLeast(5, 9, 0) && hostVer.major != 0 {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("BTRFS mount option 'rescue' requires kernel 5.9 or newer on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelWarning,
				})
			}
			rescueOpts := strings.Split(val, ":")
			for _, ropt := range rescueOpts {
				if ropt == "ignoremetacsums" || ropt == "imetacsums" {
					if !hostVer.atLeast(6, 12, 0) && hostVer.major != 0 {
						suggestions = append(suggestions, rules.Suggestion{
							Message: fmt.Sprintf("BTRFS rescue option '%s' requires kernel 6.12 or newer on line %d", ropt, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.SuggestionLevelWarning,
						})
					}
				} else if ropt != "usebackuproot" && ropt != "nologreplay" && ropt != "ignorebadroots" && ropt != "ibadroots" && ropt != "ignoredatacsums" && ropt != "idatacsums" && ropt != "all" {
					suggestions = append(suggestions, rules.Suggestion{
						Message: fmt.Sprintf("Invalid value '%s' in rescue on line %d", ropt, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.SuggestionLevelError,
					})
				}
			}
		case "compress", "compress-force":
			cparts := strings.SplitN(val, ":", 2)
			algo := cparts[0]
			if algo == "zstd" {
				if !hostVer.atLeast(5, 1, 0) && hostVer.major != 0 {
					suggestions = append(suggestions, rules.Suggestion{
						Message: fmt.Sprintf("BTRFS mount option compress algorithm 'zstd' requires kernel 5.1 or newer on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.SuggestionLevelWarning,
					})
				}
			}
			if algo != "zlib" && algo != "lzo" && algo != "zstd" && algo != "no" && algo != "" {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Invalid value '%s' for compress on line %d", algo, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
		case "discard":
			if val == "async" {
				if !hostVer.atLeast(5, 6, 0) && hostVer.major != 0 {
					suggestions = append(suggestions, rules.Suggestion{
						Message: fmt.Sprintf("BTRFS mount option 'discard=async' requires kernel 5.6 or newer on line %d", entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.SuggestionLevelWarning,
					})
				}
			} else if val != "" && val != "sync" {
				suggestions = append(suggestions, rules.Suggestion{
					Message: fmt.Sprintf("Invalid value '%s' for discard on line %d, must be 'sync' or 'async'", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.SuggestionLevelError,
				})
			}
		case "device":
			if val != "" {
				if c != nil {
					if v, ok := c.Get(rules.CacheKey("file_exists:" + val)); ok {
						if exists, ok := v.(bool); ok && !exists {
							suggestions = append(suggestions, rules.Suggestion{
								Message: fmt.Sprintf("Device '%s' specified in BTRFS mount option device does not exist on line %d", val, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.SuggestionLevelError,
							})
						}
						continue
					}
				}
				if _, err := os.Stat(val); err != nil {
					suggestions = append(suggestions, rules.Suggestion{
						Message: fmt.Sprintf("Device '%s' specified in BTRFS mount option device does not exist on line %d", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.SuggestionLevelError,
					})
				}
			}
		case "subvol":
			if val != "" {
				checked := false
				if c != nil {
					if v, ok := c.Get(rules.CacheKey("btrfs_subvol_exists:" + val)); ok {
						if exists, ok := v.(bool); ok && !exists {
							suggestions = append(suggestions, rules.Suggestion{
								Message: fmt.Sprintf("Subvolume '%s' might not exist on line %d", val, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.SuggestionLevelWarning,
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
							suggestions = append(suggestions, rules.Suggestion{
								Message: fmt.Sprintf("Subvolume '%s' might not exist on line %d", val, entry.LineNum),
								Line:    entry.RawLine,
								Level:   rules.SuggestionLevelWarning,
							})
						}
					}
				}
			}
		}
	}

	return suggestions, nil
}

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

func getHostKernelVersion(c *rules.Cache) (major, minor, patch int) {
	if c != nil {
		if v, ok := c.Get("proc_version"); ok {
			if b, ok := v.([]byte); ok {
				return parseKernelVersionStr(string(b))
			} else if s, ok := v.(string); ok {
				return parseKernelVersionStr(s)
			}
		}
		if v, ok := c.Get("proc_osrelease"); ok {
			if b, ok := v.([]byte); ok {
				return parseKernelVersionStr(string(b))
			} else if s, ok := v.(string); ok {
				return parseKernelVersionStr(s)
			}
		}
	}

	b, err := os.ReadFile("/proc/version")
	if err == nil {
		return parseKernelVersionStr(string(b))
	}
	b, err = os.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil {
		return parseKernelVersionStr(string(b))
	}
	return 0, 0, 0
}

type kernelVer struct {
	major int
	minor int
	patch int
}

func (v kernelVer) atLeast(major, minor, patch int) bool {
	if v.major > major {
		return true
	}
	if v.major == major {
		if v.minor > minor {
			return true
		}
		if v.minor == minor {
			return v.patch >= patch
		}
	}
	return false
}

var btrfsKnownOptions = map[string]struct{}{
	"acl": {}, "noacl": {}, "autodefrag": {}, "noautodefrag": {},
	"barrier": {}, "nobarrier": {}, "clear_cache": {}, "commit": {},
	"compress": {}, "compress-force": {}, "datacow": {}, "nodatacow": {},
	"datasum": {}, "nodatasum": {}, "degraded": {}, "device": {},
	"discard": {}, "nodiscard": {}, "enospc_debug": {}, "noenospc_debug": {},
	"fatal_errors": {}, "flushoncommit": {}, "noflushoncommit": {},
	"fragment": {}, "nologreplay": {}, "max_inline": {}, "metadata_ratio": {},
	"norecovery": {}, "rescan_uuid_tree": {}, "rescue": {}, "skip_balance": {},
	"space_cache": {}, "nospace_cache": {}, "ssd": {}, "ssd_spread": {},
	"nossd": {}, "nossd_spread": {}, "subvol": {}, "subvolid": {},
	"thread_pool": {}, "treelog": {}, "notreelog": {}, "usebackuproot": {},
	"user_subvol_rm_allowed": {}, "recovery": {}, "inode_cache": {},
	"noinode_cache": {}, "check_int": {}, "check_int_data": {},
	"check_int_print_mask": {},
	// Common generic ones
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},
}
