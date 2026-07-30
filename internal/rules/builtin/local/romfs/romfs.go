package romfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type RomfsRule struct{}

func init() {
	rules.Register(&RomfsRule{})
}

func (r *RomfsRule) Name() string {
	return "romfs"
}

func (r *RomfsRule) Dependencies() []string {
	return []string{}
}

func (r *RomfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "romfs" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	opts := rules.SplitOptions(entry.Fields[3])

	for _, opt := range opts {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasEquals := len(parts) > 1
		if hasEquals {
			val = parts[1]
		}

		validator, ok := romfsOptions[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown romfs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, "romfs", key, val, hasEquals, nil)...)
		}
	}

	return issues, nil
}

var romfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"loop": options.OptionalStringOption{},
})
