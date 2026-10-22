package gocryptfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type GocryptfsRule struct{}

func init() {
	rules.Register(&GocryptfsRule{})
}

func (r *GocryptfsRule) Name() string {
	return "gocryptfs"
}

func (r *GocryptfsRule) Dependencies() []string {
	return []string{}
}

func (r *GocryptfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gocryptfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "gocryptfs", gocryptfsOptions), nil
}

var gocryptfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":                options.NoValueOption{},
	"allow_other":        options.NoValueOption{},
	"badname":            options.StringOption{},
	"context":            options.StringOption{},
	"ctlsock":            options.StringOption{},
	"dev":                options.NoValueOption{},
	"nodev":              options.NoValueOption{},
	"e":                  options.StringOption{},
	"exclude":            options.StringOption{},
	"ew":                 options.StringOption{},
	"exclude-wildcard":   options.StringOption{},
	"exclude-from":       options.FileOption{CheckExistence: true},
	"exec":               options.NoValueOption{},
	"noexec":             options.NoValueOption{},
	"fg":                 options.NoValueOption{},
	"f":                  options.NoValueOption{},
	"force_owner":        ForceOwnerOption{},
	"forcedecode":        options.NoValueOption{},
	"fsname":             options.StringOption{},
	"fusedebug":          options.NoValueOption{},
	"i":                  options.StringOption{}, // duration string
	"idle":               options.StringOption{},
	"kernel_cache":       options.NoValueOption{},
	"ko":                 options.StringOption{},
	"longnames":          options.NoValueOption{},
	"nofail":             options.NoValueOption{},
	"nonempty":           options.NoValueOption{},
	"noprealloc":         options.NoValueOption{},
	"nosuid":             options.NoValueOption{},
	"nosyslog":           options.NoValueOption{},
	"notifypid":          options.IntOption{},
	"one-file-system":    options.NoValueOption{},
	"rw":                 options.NoValueOption{},
	"ro":                 options.NoValueOption{},
	"reverse":            options.NoValueOption{},
	"serialize_reads":    options.NoValueOption{},
	"sharedstorage":      options.NoValueOption{},
	"suid":               options.NoValueOption{},
	"zerokey":            options.NoValueOption{},
	"aessiv":             options.NoValueOption{}, // Not strictly a mount option but allowed by gocryptfs when used as recovery mount option
	"config":             options.FileOption{CheckExistence: true},
	"cpuprofile":         options.StringOption{},
	"d":                  options.NoValueOption{},
	"debug":              options.NoValueOption{},
	"extpass":            options.StringOption{},
	"fido2":              options.StringOption{},
	"fido2-assert-option":options.StringOption{},
	"masterkey":          options.StringOption{},
	"memprofile":         options.StringOption{},
	"o":                  options.StringOption{}, // but this is typically parsed into fstab options
	"openssl":            options.EnumOption{Allowed: map[string]bool{"true": true, "false": true, "auto": true}},
	"passfile":           options.FileOption{CheckExistence: true},
	"q":                  options.NoValueOption{},
	"quiet":              options.NoValueOption{},
	"scryptn":            options.IncorrectOption{Reason: "scryptn is an init and passwd option, not for mounting"},
	"trace":              options.StringOption{},
	"wpanic":             options.NoValueOption{},
	"-":                  options.NoValueOption{},
})
