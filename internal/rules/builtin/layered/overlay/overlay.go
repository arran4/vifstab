package overlay

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	issues := rules.ValidateOptionsMap(ctx, api, entry, "overlay", overlayOptions)

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

var overlayOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"lowerdir":     rules.StringOption{},
	"upperdir":     rules.StringOption{},
	"workdir":      rules.StringOption{},
	"userxattr":    rules.NoValueOption{},
	"redirect_dir": rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "follow": true, "nofollow": true}},
	"index":        rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"uuid":         rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"nfs_export":   rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"xino":         rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true, "auto": true}},
	"metacopy":     rules.EnumOption{Allowed: map[string]bool{"on": true, "off": true}},
	"volatile":     rules.NoValueOption{},
})
