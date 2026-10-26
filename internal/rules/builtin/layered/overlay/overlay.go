package overlay

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type OverlayRule struct{}

func init() {
	rules.Register(&OverlayRule{})
}

func (r *OverlayRule) Name() string {
	return "overlay"
}

func (r *OverlayRule) Dependencies() []string {
	return []string{}
}

func (r *OverlayRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "overlay" {
		return nil, nil
	}

	issues := options.ValidateOptionsMap(ctx, api, entry, "overlay", overlayOptions)

	// Check cross-option conflict: index=off,nfs_export=on
	options := rules.SplitOptions(entry.Fields[3])
	hasIndexOff := false
	hasNfsExportOn := false
	for _, opt := range options {
		if opt == "index=off" {
			hasIndexOff = true
		}
		if opt == "nfs_export=on" {
			hasNfsExportOn = true
		}
	}
	if hasIndexOff && hasNfsExportOn {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("overlay mount options index=off,nfs_export=on are conflicting for a read-write mount on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}

	return issues, nil
}

func unescapeFstab(s string) string {
	s = strings.ReplaceAll(s, "\\040", " ")
	s = strings.ReplaceAll(s, "\\011", "\t")
	s = strings.ReplaceAll(s, "\\012", "\n")
	s = strings.ReplaceAll(s, "\\134", "\\")
	return s
}

func parseLowerdir(val string) []string {
	var paths []string
	var currentPath strings.Builder

	for i := 0; i < len(val); i++ {
		c := val[i]
		if c == '\\' && i+1 < len(val) && val[i+1] == ':' {
			currentPath.WriteByte(':')
			i++
		} else if c == ':' {
			if i+1 < len(val) && val[i+1] == ':' {
				paths = append(paths, currentPath.String())
				currentPath.Reset()
				i++
			} else {
				paths = append(paths, currentPath.String())
				currentPath.Reset()
			}
		} else {
			currentPath.WriteByte(c)
		}
	}
	paths = append(paths, currentPath.String())
	return paths
}

func validateLowerdir(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	decodedVal := unescapeFstab(val)
	var issues []rules.ValidationIssue
	paths := parseLowerdir(decodedVal)

	for _, p := range paths {
		if p == "" {
			continue
		}
		existsResult := api.CheckFileExists(p)
		if existsResult != rules.FileExists {
			if existsResult == rules.Unknown {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unable to determine existence of file '%s' for %s mount option '%s' on line %d", p, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelInfo,
				})
				continue
			}
			if existsResult == rules.FilePermissionDenied {
				continue
			}
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("File '%s' specified in %s mount option '%s' does not exist on line %d", p, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	}

	return issues
}

var overlayOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"lowerdir":     options.OptionValidatorFunc(validateLowerdir),
	"upperdir":     options.FileOption{CheckExistence: true},
	"workdir":      options.FileOption{CheckExistence: true},
	"userxattr":    options.NoValueOption{},
	"redirect_dir": options.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "follow": true, "nofollow": true}},
	"index":        options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"uuid":         options.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "null": true, "auto": true}},
	"nfs_export":   options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"xino":         options.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "auto": true}},
	"metacopy":     options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"volatile":     options.NoValueOption{},
	"fsync":        options.EnumOption{Allowed: map[string]bool{"auto": true, "strict": true, "volatile": true}},
	"verity":       options.EnumOption{Allowed: map[string]bool{"off": true, "on": true, "require": true}},
})
