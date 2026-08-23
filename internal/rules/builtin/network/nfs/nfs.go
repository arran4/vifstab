package nfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type NfsRule struct{}

var (
	nfs3OptionsMap map[string]options.OptionValidator
	nfs4OptionsMap map[string]options.OptionValidator
)

func init() {
	rules.Register(&NfsRule{})

	baseOpts := map[string]options.OptionValidator{
		"nfsvers":     options.IntOption{},
		"vers":        options.IntOption{},
		"soft":        options.NoValueOption{},
		"softerr":     options.NoValueOption{},
		"hard":        options.NoValueOption{},
		"softreval":   options.NoValueOption{},
		"nosoftreval": options.NoValueOption{},
		"intr":        options.NoValueOption{}, // ignored after 2.6.25
		"nointr":      options.NoValueOption{},
		"timeo":       options.IntOption{},
		"retrans":     options.IntOption{},
		"rsize":       options.IntOption{},
		"wsize":       options.IntOption{},
		"ac":          options.NoValueOption{},
		"noac":        options.NoValueOption{},
		"acregmin":    options.IntOption{},
		"acregmax":    options.IntOption{},
		"acdirmin":    options.IntOption{},
		"acdirmax":    options.IntOption{},
		"actimeo":     options.IntOption{},
		"bg":          options.NoValueOption{},
		"fg":          options.NoValueOption{},
		"nconnect":    options.IntRangeOption{Max: func() *int64 { i := int64(16); return &i }()}, // limit of 16
		"rdirplus": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
			if !hasVal {
				return nil
			}
			if val != "none" && val != "force" {
				return []rules.ValidationIssue{
					{
						Message: fmt.Sprintf("Invalid value '%s' for %s mount option 'rdirplus' on line %d", val, fsName, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					},
				}
			}
			return nil
		}),
		"nordirplus":   options.NoValueOption{},
		"retry":        options.IntOption{},
		"sec":          NfsSecOption{},
		"sharecache":   options.NoValueOption{},
		"nosharecache": options.NoValueOption{},
		"resvport":     options.NoValueOption{},
		"noresvport":   options.NoValueOption{},
		"lookupcache":  options.EnumOption{Allowed: map[string]bool{"all": true, "none": true, "pos": true, "positive": true}},
		"fsc":          options.NoValueOption{},
		"nofsc":        options.NoValueOption{},
		"sloppy":       options.NoValueOption{},
		"xprtsec":      options.EnumOption{Allowed: map[string]bool{"none": true, "tls": true, "mtls": true}},
		"noalignwrite": options.NoValueOption{},

		// NFS version 4 specific options (accepted by both nfs and nfs4 for compatibility)
		"minorversion":     options.IntOption{},
		"clientaddr":       options.StringOption{},
		"migration":        options.NoValueOption{},
		"nomigration":      options.NoValueOption{},
		"max_connect":      options.IntRangeOption{Max: func() *int64 { i := int64(16); return &i }()}, // limit of 16
		"trunkdiscovery":   options.NoValueOption{},
		"notrunkdiscovery": options.NoValueOption{},

		// exports(5) options mapped as incorrectly used
		"all_squash":       options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"anongid":          options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"anonuid":          options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"crossmnt":         options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"fsid":             options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"hide":             options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"insecure_locks":   options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"mountpoint":       options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"mp":               options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"no_auth_nlm":      options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"no_root_squash":   options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"no_subtree_check": options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"no_wdelay":        options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"nocrossmnt":       options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"nohide":           options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"pnfs":             options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"reexport":         options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"refer":            options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"replicas":         options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"root_squash":      options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"secure":           options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"security_label":   options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"sign_fh":          options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
		"subtree_check":    options.IncorrectOption{Reason: "this is an exports(5) option used for NFS server configuration, not for client mounts"},
	}

	nfs3Specific := map[string]options.OptionValidator{
		"proto":      options.EnumOption{Allowed: map[string]bool{"udp": true, "udp6": true, "tcp": true, "tcp6": true, "rdma": true, "rdma6": true}},
		"udp":        options.NoValueOption{},
		"tcp":        options.NoValueOption{},
		"rdma":       options.NoValueOption{},
		"port":       options.IntOption{},
		"mountport":  options.IntOption{},
		"mountproto": options.EnumOption{Allowed: map[string]bool{"udp": true, "udp6": true, "tcp": true, "tcp6": true}},
		"mounthost":  options.StringOption{},
		"mountvers":  options.IntOption{},
		"namlen":     options.IntOption{},
		"lock":       options.NoValueOption{},
		"nolock":     options.NoValueOption{},
		"cto":        options.NoValueOption{},
		"nocto":      options.NoValueOption{},
		"acl":        options.NoValueOption{},
		"noacl":      options.NoValueOption{},
		"local_lock": options.EnumOption{Allowed: map[string]bool{"all": true, "flock": true, "posix": true, "none": true}},
	}

	nfs4Specific := map[string]options.OptionValidator{
		"proto": options.EnumOption{Allowed: map[string]bool{"tcp": true, "tcp6": true, "rdma": true, "rdma6": true}},
		"port":  options.IntOption{},
		"cto":   options.NoValueOption{},
		"nocto": options.NoValueOption{},
	}

	nfs3OptionsMap = options.MergeOptions(options.CommonOptions(), baseOpts, nfs3Specific)
	nfs4OptionsMap = options.MergeOptions(options.CommonOptions(), baseOpts, nfs4Specific)
}

func (r *NfsRule) Name() string {
	return "nfs"
}

func (r *NfsRule) Dependencies() []string {
	return []string{}
}

type NfsSecOption struct{}

func (o NfsSecOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	flavors := strings.Split(val, ":")
	validFlavors := map[string]bool{
		"none":  true,
		"sys":   true,
		"krb5":  true,
		"krb5i": true,
		"krb5p": true,
	}
	for _, flavor := range flavors {
		if !validFlavors[flavor] {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}
	return nil
}

func (r *NfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nfs" && fsType != "nfs4" {
		return nil, nil
	}

	var allOpts map[string]options.OptionValidator
	if fsType == "nfs4" {
		allOpts = nfs4OptionsMap
	} else {
		allOpts = nfs3OptionsMap
	}

	issues := options.ValidateOptionsMap(ctx, api, entry, fsType, allOpts)
	return issues, nil
}
