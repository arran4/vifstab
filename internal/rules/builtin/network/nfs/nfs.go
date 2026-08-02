package nfs

import (
	"context"
	"strings"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type NfsRule struct{}

func init() {
	rules.Register(&NfsRule{})
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
		"none": true,
		"sys":  true,
		"krb5": true,
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


func nfsOptions(isNfs4 bool) map[string]options.OptionValidator {
	opts := map[string]options.OptionValidator{
		"nfsvers":      options.IntOption{},
		"vers":         options.IntOption{},
		"soft":         options.NoValueOption{},
		"softerr":      options.NoValueOption{},
		"hard":         options.NoValueOption{},
		"softreval":    options.NoValueOption{},
		"nosoftreval":  options.NoValueOption{},
		"intr":         options.NoValueOption{}, // ignored after 2.6.25
		"nointr":       options.NoValueOption{},
		"timeo":        options.IntOption{},
		"retrans":      options.IntOption{},
		"rsize":        options.IntOption{},
		"wsize":        options.IntOption{},
		"ac":           options.NoValueOption{},
		"noac":         options.NoValueOption{},
		"acregmin":     options.IntOption{},
		"acregmax":     options.IntOption{},
		"acdirmin":     options.IntOption{},
		"acdirmax":     options.IntOption{},
		"actimeo":      options.IntOption{},
		"bg":           options.NoValueOption{},
		"fg":           options.NoValueOption{},
		"nconnect":     options.IntRangeOption{Max: func() *int64 { i := int64(16); return &i }()}, // limit of 16
		"rdirplus":     options.NoValueOption{}, // can be boolean or force/none string
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

		// NFS version 4 specific options
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

	if isNfs4 {
		opts["proto"] = options.EnumOption{Allowed: map[string]bool{"tcp": true, "tcp6": true, "rdma": true, "rdma6": true}}
		opts["port"] = options.IntOption{}
		opts["cto"] = options.NoValueOption{}
		opts["nocto"] = options.NoValueOption{}
	} else {
		// NFS versions 2 and 3 specific options
		opts["proto"] = options.EnumOption{Allowed: map[string]bool{"udp": true, "udp6": true, "tcp": true, "tcp6": true, "rdma": true, "rdma6": true}}
		opts["udp"] = options.NoValueOption{}
		opts["tcp"] = options.NoValueOption{}
		opts["rdma"] = options.NoValueOption{}
		opts["port"] = options.IntOption{}
		opts["mountport"] = options.IntOption{}
		opts["mountproto"] = options.EnumOption{Allowed: map[string]bool{"udp": true, "udp6": true, "tcp": true, "tcp6": true}}
		opts["mounthost"] = options.StringOption{}
		opts["mountvers"] = options.IntOption{}
		opts["namlen"] = options.IntOption{}
		opts["lock"] = options.NoValueOption{}
		opts["nolock"] = options.NoValueOption{}
		opts["cto"] = options.NoValueOption{}
		opts["nocto"] = options.NoValueOption{}
		opts["acl"] = options.NoValueOption{}
		opts["noacl"] = options.NoValueOption{}
		opts["local_lock"] = options.EnumOption{Allowed: map[string]bool{"all": true, "flock": true, "posix": true, "none": true}}
	}

	// rdirplus can be boolean or {none|force}
	opts["rdirplus"] = options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
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
	})

	return opts
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

	optsMap := nfsOptions(fsType == "nfs4")
	allOpts := options.MergeOptions(options.CommonOptions(), optsMap)

	issues := options.ValidateOptionsMap(ctx, api, entry, fsType, allOpts)
	return issues, nil
}
