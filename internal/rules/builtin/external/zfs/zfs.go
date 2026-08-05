package zfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ZFSRule struct{}

func init() {
	rules.Register(&ZFSRule{})
}

func (r *ZFSRule) Name() string {
	return "zfs"
}

func (r *ZFSRule) Dependencies() []string {
	return []string{}
}

func (r *ZFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "zfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "zfs", "")

	entryCopy := *entry
	entryCopy.Fields = append([]string(nil), entry.Fields...)

	opts := rules.SplitOptions(entry.Fields[3])
	var filteredOpts []string

	for _, opt := range opts {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]

		// Handle dynamic options
		if strings.HasPrefix(key, "userused@") ||
			strings.HasPrefix(key, "userobjused@") ||
			strings.HasPrefix(key, "groupused@") ||
			strings.HasPrefix(key, "groupobjused@") ||
			strings.HasPrefix(key, "projectused@") ||
			strings.HasPrefix(key, "projectobjused@") ||
			strings.HasPrefix(key, "userquota@") ||
			strings.HasPrefix(key, "userobjquota@") ||
			strings.HasPrefix(key, "groupquota@") ||
			strings.HasPrefix(key, "groupobjquota@") ||
			strings.HasPrefix(key, "projectquota@") ||
			strings.HasPrefix(key, "projectobjquota@") ||
			strings.HasPrefix(key, "written@") ||
			strings.Contains(key, ":") { // Handle user properties like module:property
			continue
		}

		filteredOpts = append(filteredOpts, opt)
	}

	if len(filteredOpts) > 0 {
		entryCopy.Fields[3] = strings.Join(filteredOpts, ",")
	} else {
		entryCopy.Fields[3] = ""
	}

	optsIssues := options.ValidateOptionsMap(ctx, api, &entryCopy, fsType, zfsOptions)
	issues = append(issues, optsIssues...)

	return issues, nil
}
