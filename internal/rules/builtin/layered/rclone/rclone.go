package rclone

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

var rcloneOptions = map[string]options.OptionValidator{
	"config":                    options.StringOption{},
	"args2env":                  options.NoValueOption{},
	"command":                   options.StringOption{},
	"allow-idmap":               options.NoValueOption{},
	"allow-non-empty":           options.NoValueOption{},
	"allow-other":               options.NoValueOption{},
	"allow-root":                options.NoValueOption{},
	"allow_idmap":               options.NoValueOption{},
	"allow_non_empty":           options.NoValueOption{},
	"allow_other":               options.NoValueOption{},
	"allow_root":                options.NoValueOption{},
	"async-read":                options.BoolOption{},
	"async_read":                options.BoolOption{},
	"attr-timeout":              options.StringOption{},
	"attr_timeout":              options.StringOption{},
	"cache-dir":                 options.StringOption{},
	"cache_dir":                 options.StringOption{},
	"daemon":                    options.NoValueOption{},
	"daemon-timeout":            options.StringOption{},
	"daemon-wait":               options.StringOption{},
	"daemon_timeout":            options.StringOption{},
	"daemon_wait":               options.StringOption{},
	"debug-fuse":                options.NoValueOption{},
	"debug_fuse":                options.NoValueOption{},
	"default-permissions":       options.NoValueOption{},
	"default_permissions":       options.NoValueOption{},
	"delete-excluded":           options.NoValueOption{},
	"delete_excluded":           options.NoValueOption{},
	"devname":                   options.StringOption{},
	"dir-cache-time":            options.StringOption{},
	"dir-perms":                 options.UmaskOption{},
	"dir_cache_time":            options.StringOption{},
	"dir_perms":                 options.UmaskOption{},
	"direct-io":                 options.NoValueOption{},
	"direct_io":                 options.NoValueOption{},
	"exclude":                   options.StringOption{},
	"exclude-from":              options.StringOption{},
	"exclude-if-present":        options.StringOption{},
	"exclude_from":              options.StringOption{},
	"exclude_if_present":        options.StringOption{},
	"file-perms":                options.UmaskOption{},
	"file_perms":                options.UmaskOption{},
	"files-from":                options.StringOption{},
	"files-from-raw":            options.StringOption{},
	"files-from0":               options.StringOption{},
	"files_from":                options.StringOption{},
	"files_from0":               options.StringOption{},
	"files_from_raw":            options.StringOption{},
	"filter-from":               options.StringOption{},
	"filter_from":               options.StringOption{},
	"fuse-flag":                 options.StringOption{},
	"fuse_flag":                 options.StringOption{},
	"gid":                       options.GIDOption{},
	"hash-filter":               options.StringOption{},
	"hash_filter":               options.StringOption{},
	"ignore-case":               options.NoValueOption{},
	"ignore_case":               options.NoValueOption{},
	"include":                   options.StringOption{},
	"include-from":              options.StringOption{},
	"include_from":              options.StringOption{},
	"link-perms":                options.UmaskOption{},
	"link_perms":                options.UmaskOption{},
	"links":                     options.NoValueOption{},
	"max-age":                   options.StringOption{},
	"max-depth":                 options.IntOption{},
	"max-read-ahead":            options.StringOption{},
	"max-size":                  options.StringOption{},
	"max_age":                   options.StringOption{},
	"max_depth":                 options.IntOption{},
	"max_read_ahead":            options.StringOption{},
	"max_size":                  options.StringOption{},
	"metadata-exclude":          options.StringOption{},
	"metadata-exclude-from":     options.StringOption{},
	"metadata-filter":           options.StringOption{},
	"metadata-filter-from":      options.StringOption{},
	"metadata-include":          options.StringOption{},
	"metadata-include-from":     options.StringOption{},
	"metadata_exclude":          options.StringOption{},
	"metadata_exclude_from":     options.StringOption{},
	"metadata_filter":           options.StringOption{},
	"metadata_filter_from":      options.StringOption{},
	"metadata_include":          options.StringOption{},
	"metadata_include_from":     options.StringOption{},
	"min-age":                   options.StringOption{},
	"min-size":                  options.StringOption{},
	"min_age":                   options.StringOption{},
	"min_size":                  options.StringOption{},
	"mount-case-insensitive":    options.BoolOption{},
	"mount_case_insensitive":    options.BoolOption{},
	"network-mode":              options.NoValueOption{},
	"network_mode":              options.NoValueOption{},
	"no-checksum":               options.NoValueOption{},
	"no-modtime":                options.NoValueOption{},
	"no-seek":                   options.NoValueOption{},
	"no_checksum":               options.NoValueOption{},
	"no_modtime":                options.NoValueOption{},
	"no_seek":                   options.NoValueOption{},
	"noappledouble":             options.BoolOption{},
	"noapplexattr":              options.NoValueOption{},
	"poll-interval":             options.StringOption{},
	"poll_interval":             options.StringOption{},
	"read-only":                 options.NoValueOption{},
	"read_only":                 options.NoValueOption{},
	"transfers":                 options.IntOption{},
	"uid":                       options.UIDOption{},
	"umask":                     options.UmaskOption{},
	"vfs-block-norm-dupes":      options.NoValueOption{},
	"vfs-cache-max-age":         options.StringOption{},
	"vfs-cache-max-size":        options.StringOption{},
	"vfs-cache-min-free-space":  options.StringOption{},
	"vfs-cache-mode":            options.EnumOption{Allowed: map[string]bool{"off": true, "minimal": true, "writes": true, "full": true}},
	"vfs-cache-poll-interval":   options.StringOption{},
	"vfs-case-insensitive":      options.NoValueOption{},
	"vfs-disk-space-total-size": options.StringOption{},
	"vfs-fast-fingerprint":      options.NoValueOption{},
	"vfs-handle-caching":        options.StringOption{},
	"vfs-links":                 options.NoValueOption{},
	"vfs-metadata-extension":    options.StringOption{},
	"vfs-read-ahead":            options.StringOption{},
	"vfs-read-chunk-size":       options.StringOption{},
	"vfs-read-chunk-size-limit": options.StringOption{},
	"vfs-read-chunk-streams":    options.IntOption{},
	"vfs-read-wait":             options.StringOption{},
	"vfs-refresh":               options.NoValueOption{},
	"vfs-used-is-size":          options.NoValueOption{},
	"vfs-write-back":            options.StringOption{},
	"vfs-write-wait":            options.StringOption{},
	"vfs_block_norm_dupes":      options.NoValueOption{},
	"vfs_cache_max_age":         options.StringOption{},
	"vfs_cache_max_size":        options.StringOption{},
	"vfs_cache_min_free_space":  options.StringOption{},
	"vfs_cache_mode":            options.EnumOption{Allowed: map[string]bool{"off": true, "minimal": true, "writes": true, "full": true}},
	"vfs_cache_poll_interval":   options.StringOption{},
	"vfs_case_insensitive":      options.NoValueOption{},
	"vfs_disk_space_total_size": options.StringOption{},
	"vfs_fast_fingerprint":      options.NoValueOption{},
	"vfs_handle_caching":        options.StringOption{},
	"vfs_links":                 options.NoValueOption{},
	"vfs_metadata_extension":    options.StringOption{},
	"vfs_read_ahead":            options.StringOption{},
	"vfs_read_chunk_size":       options.StringOption{},
	"vfs_read_chunk_size_limit": options.StringOption{},
	"vfs_read_chunk_streams":    options.IntOption{},
	"vfs_read_wait":             options.StringOption{},
	"vfs_refresh":               options.NoValueOption{},
	"vfs_used_is_size":          options.NoValueOption{},
	"vfs_write_back":            options.StringOption{},
	"vfs_write_wait":            options.StringOption{},
	"volname":                   options.StringOption{},
	"write-back-cache":          options.NoValueOption{},
	"write_back_cache":          options.NoValueOption{},
}

type RcloneRule struct{}

func init() {
	rules.Register(&RcloneRule{})
}

func (r *RcloneRule) Name() string {
	return "rclone"
}

func (r *RcloneRule) Dependencies() []string {
	return []string{}
}

func (r *RcloneRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "rclone" {
		return nil, nil
	}

	originalOpts := entry.Fields[3]
	defer func() { entry.Fields[3] = originalOpts }()

	var filteredOpts []string
	for _, opt := range rules.SplitOptions(originalOpts) {
		key := strings.SplitN(opt, "=", 2)[0]
		if strings.HasPrefix(key, "env.") || key == "v" || key == "vv" || key == "vvv" {
			continue
		}
		filteredOpts = append(filteredOpts, opt)
	}
	entry.Fields[3] = strings.Join(filteredOpts, ",")

	mergedOptions := options.MergeOptions(options.CommonOptions(), rcloneOptions)
	issues := options.ValidateOptionsMap(ctx, api, entry, "rclone", mergedOptions)

	return issues, nil
}
