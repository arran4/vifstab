package sshfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type SshfsRule struct{}

func init() {
	rules.Register(&SshfsRule{})
}

func (r *SshfsRule) Name() string {
	return "sshfs"
}

func (r *SshfsRule) Dependencies() []string {
	return []string{}
}

func (r *SshfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "sshfs" && fsType != "fuse.sshfs" {
		return nil, nil
	}

	fuseOptions := map[string]options.OptionValidator{
		"default_permissions": options.NoValueOption{},
		"allow_other":         options.NoValueOption{},
		"rootmode":            options.UmaskOption{},
		"blkdev":              options.NoValueOption{},
		"blksize":             options.IntOption{},
		"max_read":            options.SizeOption{},
		"fd":                  options.IntOption{},
		"user_id":             options.UIDOption{},
		"group_id":            options.GIDOption{},
		"fsname":              options.StringOption{},
		"subtype":             options.StringOption{},
		"allow_root":          options.NoValueOption{},
		"auto_unmount":        options.NoValueOption{},
		"kernel_cache":        options.NoValueOption{},
		"auto_cache":          options.NoValueOption{},
		"umask":               options.UmaskOption{},
		"fmask":               options.UmaskOption{},
		"dmask":               options.UmaskOption{},
		"uid":                 options.UIDOption{},
		"gid":                 options.GIDOption{},
		"entry_timeout":       options.StringOption{},
		"negative_timeout":    options.StringOption{},
		"attr_timeout":        options.StringOption{},
		"ac_attr_timeout":     options.StringOption{},
		"noforget":            options.NoValueOption{},
		"remember":            options.StringOption{},
		"modules":             options.StringOption{},
		"setuid":              options.StringOption{},
		"drop_privileges":     options.NoValueOption{},
		"from_code":           options.IOCharsetOption{},
		"to_code":             options.IOCharsetOption{},
		"subdir":              options.StringOption{},
		"rellinks":            options.NoValueOption{},
		"norellinks":          options.NoValueOption{},
	}

	sshfsOptions := map[string]options.OptionValidator{
		"port":                      options.IntOption{},
		"reconnect":                 options.NoValueOption{},
		"delay_connect":             options.NoValueOption{},
		"sshfs_sync":                options.NoValueOption{},
		"no_readahead":              options.NoValueOption{},
		"sync_readdir":              options.NoValueOption{},
		"workaround":                options.OptionValidatorFunc(validateWorkaround),
		"idmap":                     options.EnumOption{Allowed: map[string]bool{"none": true, "user": true, "file": true}},
		"uidfile":                   options.StringOption{},
		"gidfile":                   options.StringOption{},
		"nomap":                     options.EnumOption{Allowed: map[string]bool{"ignore": true, "error": true}},
		"ssh_command":               options.StringOption{},
		"ssh_protocol":              options.IntOption{},
		"sftp_server":               options.StringOption{},
		"directport":                options.IntOption{},
		"passive":                   options.NoValueOption{},
		"disable_hardlink":          options.NoValueOption{},
		"transform_symlinks":        options.NoValueOption{},
		"contain_symlinks":          options.NoValueOption{},
		"no_contain_symlinks":       options.NoValueOption{},
		"follow_symlinks":           options.NoValueOption{},
		"no_check_root":             options.NoValueOption{},
		"password_stdin":            options.NoValueOption{},
		"dir_cache":                 options.EnumOption{Allowed: map[string]bool{"yes": true, "no": true}},
		"dcache_max_size":           options.IntOption{},
		"dcache_timeout":            options.IntOption{},
		"dcache_stat_timeout":       options.IntOption{},
		"dcache_link_timeout":       options.IntOption{},
		"dcache_dir_timeout":        options.IntOption{},
		"dcache_clean_interval":     options.IntOption{},
		"dcache_min_clean_interval": options.IntOption{},
		"direct_io":                 options.NoValueOption{},
		"max_conns":                 options.IntOption{},
		"compression":               options.EnumOption{Allowed: map[string]bool{"yes": true, "no": true}},
	}

	fsOpts := options.MergeOptions(options.CommonOptions(), fuseOptions, sshfsOptions)

	var filteredOpts []string
	parsedOpts := rules.SplitOptions(entry.Fields[3])
	for _, opt := range parsedOpts {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		if len(key) > 0 && key[0] >= 'A' && key[0] <= 'Z' && !strings.HasPrefix(key, "X-") {
			// Skip capitalized options as they are passed to ssh (e.g., ServerAliveInterval)
			continue
		}
		filteredOpts = append(filteredOpts, opt)
	}

	entryCopy := *entry
	entryCopy.Fields = make([]string, len(entry.Fields))
	copy(entryCopy.Fields, entry.Fields)
	entryCopy.Fields[3] = strings.Join(filteredOpts, ",")

	return options.ValidateOptionsMap(ctx, api, &entryCopy, fsType, fsOpts), nil
}

func validateWorkaround(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	valid := map[string]bool{
		"rename":     true,
		"renamexdev": true,
		"truncate":   true,
		"fstat":      true,
		"buflimit":   true,
		"createmode": true,
	}

	parts := strings.Split(val, ":")
	for _, p := range parts {
		if !valid[p] {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", p, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}

	return nil
}
