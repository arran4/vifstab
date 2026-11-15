package mergerfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type MergerfsRule struct{}

func init() {
	rules.Register(&MergerfsRule{})
}

func (r *MergerfsRule) Name() string {
	return "mergerfs"
}

func (r *MergerfsRule) Dependencies() []string {
	return []string{}
}

var mergerfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"config":                      options.StringOption{},
	"branches":                    options.StringOption{},
	"mountpoint":                  options.StringOption{},
	"branches-mount-timeout":      options.IntOption{},
	"branches-mount-timeout-fail": options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"minfreespace":                options.StringOption{},
	"moveonenospc":                options.StringOption{},
	"inodecalc":                   options.EnumOption{Allowed: map[string]bool{"passthrough": true, "path-hash": true, "devino-hash": true, "hybrid-hash": true}},
	"dropcacheonclose":            options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"direct-io-allow-mmap":        options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"symlinkify":                  options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"symlinkify-timeout":          options.IntOption{},
	"nullrw":                      options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"lazy-umount-mountpoint":      options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"ignorepponrename":            options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"export-support":              options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"kernel-permissions-check":    options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"security-capability":         options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"xattr":                       options.EnumOption{Allowed: map[string]bool{"passthrough": true, "noattr": true, "nosys": true}},
	"link-cow":                    options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"statfs":                      options.EnumOption{Allowed: map[string]bool{"base": true, "full": true}},
	"statfs-ignore":               options.EnumOption{Allowed: map[string]bool{"none": true, "ro": true, "nc": true}},
	"nfsopenhack":                 options.EnumOption{Allowed: map[string]bool{"off": true, "git": true, "all": true}},
	"follow-symlinks":             options.EnumOption{Allowed: map[string]bool{"never": true, "directory": true, "regular": true, "all": true}},
	"link-exdev":                  options.EnumOption{Allowed: map[string]bool{"passthrough": true, "rel-symlink": true, "abs-base-symlink": true, "abs-pool-symlink": true}},
	"rename-exdev":                options.EnumOption{Allowed: map[string]bool{"passthrough": true, "rel-symlink": true, "abs-symlink": true}},
	"readahead":                   options.IntOption{},
	"posix-acl":                   options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"async-read":                  options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"fuse-msg-size":               options.StringOption{},
	"read-thread-count":           options.IntOption{},
	"process-thread-count":        options.IntOption{},
	"process-thread-queue-depth":  options.IntOption{},
	"pin-threads":                 options.StringOption{},
	"flush-on-close":              options.EnumOption{Allowed: map[string]bool{"never": true, "always": true, "opened-for-write": true}},
	"scheduling-priority":         options.NewIntRange(-20, 19),
	"proxy-ioprio":                options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"fsname":                      options.StringOption{},

	"cache.statfs":              options.IntOption{},
	"cache.attr":                options.IntOption{},
	"cache.entry":               options.IntOption{},
	"cache.negative-entry":      options.IntOption{},
	"cache.files":               options.EnumOption{Allowed: map[string]bool{"off": true, "partial": true, "full": true, "auto-full": true, "per-process": true}},
	"cache.files.process-names": options.StringOption{},
	"cache.writeback":           options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"cache.symlinks":            options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"cache.readdir":             options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},

	"parallel-direct-writes":      options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"passthrough.io":              options.StringOption{},
	"passthrough.max-stack-depth": options.IntOption{},
	"never-forget-nodes":          options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"allow-idmap":                 options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"debug":                       options.EnumOption{Allowed: map[string]bool{"true": true, "false": true}},
	"log.file":                    options.StringOption{},
})

func (r *MergerfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "mergerfs" {
		return nil, nil
	}

	entryCopy := *entry
	entryCopy.Fields = append([]string(nil), entry.Fields...)

	opts := rules.SplitOptions(entry.Fields[3])
	var filteredOpts []string

	for _, opt := range opts {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, "func.") || strings.HasPrefix(key, "category.") {
			continue
		}
		filteredOpts = append(filteredOpts, opt)
	}

	if len(filteredOpts) > 0 {
		entryCopy.Fields[3] = strings.Join(filteredOpts, ",")
	} else {
		entryCopy.Fields[3] = ""
	}

	return options.ValidateOptionsMap(ctx, api, &entryCopy, fsType, mergerfsOptions), nil
}
