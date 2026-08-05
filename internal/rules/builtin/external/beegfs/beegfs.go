package beegfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
	"github.com/arran4/vifstab/internal/rules/options"
)

var beegfsOptionsMap = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"cfgFile":                   options.StringOption{},
	"logStdFile":                options.StringOption{},
	"sysMgmtdHost":              options.StringOption{},
	"tunePreferredMetaFile":     options.StringOption{},
	"tunePreferredStorageFile":  options.StringOption{},
	"connInterfacesList":        options.StringOption{},
	"connAuthFile":              options.StringOption{},
	"connDisableAuthentication": options.StringOption{},
	"connDisableIPv6":           options.StringOption{},
	"logLevel":                  options.IntOption{},
	"connPortShift":             options.IntOption{},
	"connMgmtdPort":             options.IntOption{},
	"sysMountSanityCheckMS":     options.IntOption{},
	"grpid":                     options.NoValueOption{},
	"connMaxInternodeNum":       options.IntOption{},
	"connRDMABufSize":           options.IntOption{},
	"connRDMABufNum":            options.IntOption{},
	"tuneRemoteFSync":           options.StringOption{},
	"connCommRetrySecs":         options.IntOption{},
})

type BeeGFSRule struct{}

func init() {
	rules.Register(&BeeGFSRule{})
}

func (r *BeeGFSRule) Name() string {
	return "beegfs"
}

func (r *BeeGFSRule) Dependencies() []string {
	return []string{}
}

func (r *BeeGFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "beegfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "beegfs", "")

	optIssues := options.ValidateOptionsMap(ctx, api, entry, "beegfs", beegfsOptionsMap)
	issues = append(issues, optIssues...)

	return issues, nil
}
