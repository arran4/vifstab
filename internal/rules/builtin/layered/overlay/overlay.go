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

var overlayOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"lowerdir":     options.StringOption{},
	"upperdir":     options.StringOption{},
	"workdir":      options.StringOption{},
	"userxattr":    options.NoValueOption{},
	"redirect_dir": options.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "follow": true, "nofollow": true}},
	"index":        options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"uuid":         options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"nfs_export":   options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"xino":         options.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "auto": true}},
	"metacopy":     options.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"volatile":     options.NoValueOption{},
})
