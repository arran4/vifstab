package hostfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type HostfsRule struct{}

func init() {
	rules.Register(&HostfsRule{})
}

func (r *HostfsRule) Name() string {
	return "hostfs"
}

func (r *HostfsRule) Dependencies() []string {
	return []string{}
}

func (r *HostfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hostfs" {
		return nil, nil
	}

	issues := options.ValidateOptionsMap(ctx, api, entry, "hostfs", hostfsOptions)

	dev := entry.Fields[0]
	if strings.HasPrefix(dev, "/dev/") && !strings.HasPrefix(dev, "/dev/ubd") {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("Device for hostfs usually should be 'none' or a host path on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelWarning,
		})
	}

	return issues, nil
}

var hostfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"append": options.NoValueOption{},
})
