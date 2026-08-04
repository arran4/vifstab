package s3fs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type S3fsRule struct{}

func init() {
	rules.Register(&S3fsRule{})
}

func (r *S3fsRule) Name() string {
	return "s3fs"
}

func (r *S3fsRule) Dependencies() []string {
	return []string{}
}

func (r *S3fsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "s3fs" {
		return nil, nil
	}

	fsOpts := options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
		"bucket":                     options.StringOption{},
		"default_acl":                options.StringOption{},
		"retries":                    options.IntOption{},
		"tmpdir":                     options.StringOption{},
		"use_cache":                  options.OptionalStringOption{},
		"check_cache_dir_exist":      options.NoValueOption{},
		"del_cache":                  options.NoValueOption{},
		"storage_class":              options.EnumOption{Allowed: map[string]bool{"standard": true, "standard_ia": true, "onezone_ia": true, "reduced_redundancy": true, "intelligent_tiering": true, "glacier": true, "glacier_ir": true, "deep_archive": true}},
		"use_rrs":                    options.OptionalStringOption{},
		"use_sse":                    options.OptionalStringOption{},
		"load_sse_c":                 options.StringOption{},
		"passwd_file":                options.StringOption{},
		"ahbe_conf":                  options.StringOption{},
		"profile":                    options.StringOption{},
		"public_bucket":              options.OptionalStringOption{},
		"connect_timeout":            options.IntOption{},
		"readwrite_timeout":          options.IntOption{},
		"list_object_max_keys":       options.IntOption{},
		"max_stat_cache_size":        options.IntOption{},
		"stat_cache_expire":          options.IntOption{},
		"stat_cache_interval_expire": options.IntOption{},
		"disable_noobj_cache":        options.NoValueOption{},
		"no_check_certificate":       options.NoValueOption{},
		"ssl_verify_hostname":        options.IntOption{},
		"ssl_client_cert":            options.StringOption{},
		"nodnscache":                 options.NoValueOption{},
		"nosscache":                  options.NoValueOption{},
		"multireq_max":               options.IntOption{},
		"parallel_count":             options.IntOption{},
		"multipart_size":             options.IntOption{},
		"multipart_copy_size":        options.IntOption{},
		"max_dirty_data":             options.IntOption{},
		"bucket_size":                options.StringOption{},
		"ensure_diskfree":            options.IntOption{},
		"free_space_ratio":           options.IntOption{},
		"multipart_threshold":        options.IntOption{},
		"singlepart_copy_limit":      options.IntOption{},
		"host":                       options.StringOption{},
		"servicepath":                options.StringOption{},
		"url":                        options.StringOption{},
		"endpoint":                   options.StringOption{},
		"sigv2":                      options.NoValueOption{},
		"sigv4":                      options.NoValueOption{},
		"mp_umask":                   options.UmaskOption{},
		"umask":                      options.UmaskOption{},
		"nomultipart":                options.NoValueOption{},
		"streamupload":               options.NoValueOption{},
		"max_thread_count":           options.IntOption{},
		"enable_content_md5":         options.NoValueOption{},
		"enable_unsigned_payload":    options.NoValueOption{},
		"ecs":                        options.NoValueOption{},
		"iam_role":                   options.OptionalStringOption{},
		"imdsv1only":                 options.NoValueOption{},
		"ibm_iam_auth":               options.NoValueOption{},
		"ibm_iam_endpoint":           options.StringOption{},
		"credlib":                    options.StringOption{},
		"credlib_opts":               options.StringOption{},
		"use_xattr":                  options.NoValueOption{},
		"noxmlns":                    options.NoValueOption{},
		"nomixupload":                options.NoValueOption{},
		"nocopyapi":                  options.NoValueOption{},
		"norenameapi":                options.NoValueOption{},
		"use_path_request_style":     options.NoValueOption{},
		"listobjectsv2":              options.NoValueOption{},
		"noua":                       options.NoValueOption{},
		"cipher_suites":              options.StringOption{},
		"instance_name":              options.StringOption{},
		"complement_stat":            options.NoValueOption{},
		"compat_dir":                 options.NoValueOption{},
		"use_wtf8":                   options.NoValueOption{},
		"use_session_token":          options.NoValueOption{},
		"requester_pays":             options.NoValueOption{},
		"mime":                       options.StringOption{},
		"proxy":                      options.StringOption{},
		"proxy_cred_file":            options.StringOption{},
		"ipresolve":                  options.EnumOption{Allowed: map[string]bool{"whatever": true, "IPv4": true, "IPv6": true}},
		"logfile":                    options.StringOption{},
		"dbglevel":                   options.EnumOption{Allowed: map[string]bool{"crit": true, "err": true, "warn": true, "info": true}},
		"curldbg":                    options.OptionalStringOption{},
		"no_time_stamp_msg":          options.NoValueOption{},
		"set_check_cache_sigusr1":    options.OptionalStringOption{},
		"update_parent_dir_stat":     options.NoValueOption{},

		// Standard FUSE Options commonly used with s3fs
		"allow_other":         options.NoValueOption{},
		"allow_root":          options.NoValueOption{},
		"nonempty":            options.NoValueOption{},
		"default_permissions": options.NoValueOption{},
		"uid":                 options.UIDOption{},
		"gid":                 options.GIDOption{},
		"use_ino":             options.NoValueOption{},
		"kernel_cache":        options.NoValueOption{},
		"auto_cache":          options.NoValueOption{},
	})

	return options.ValidateOptionsMap(ctx, api, entry, fsType, fsOpts), nil
}
