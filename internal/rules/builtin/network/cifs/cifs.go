package cifs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type CifsRule struct{}

func init() {
	rules.Register(&CifsRule{})
}

func (r *CifsRule) Name() string {
	return "cifs"
}

func (r *CifsRule) Dependencies() []string {
	return []string{}
}

func (r *CifsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cifs" {
		return nil, nil
	}

	fsOptions := map[string]options.OptionValidator{
		"username":            options.StringOption{},
		"user":                options.StringOption{},
		"password":            options.StringOption{},
		"pass":                options.StringOption{},
		"password2":           options.StringOption{},
		"pass2":               options.StringOption{},
		"credentials":         options.StringOption{},
		"cred":                options.StringOption{},
		"cruid":               options.StringOption{},
		"port":                options.StringOption{},
		"netbiosname":         options.StringOption{},
		"servern":             options.StringOption{},
		"ip":                  options.StringOption{},
		"addr":                options.StringOption{},
		"domain":              options.StringOption{},
		"dom":                 options.StringOption{},
		"workgroup":           options.StringOption{},
		"snapshot":            options.StringOption{},
		"backupuid":           options.StringOption{},
		"backupgid":           options.StringOption{},
		"iocharset":           options.IOCharsetOption{},
		"uid":                 options.UIDOption{},
		"gid":                 options.GIDOption{},
		"file_mode":           options.UmaskOption{},
		"dir_mode":            options.UmaskOption{},
		"cache":               options.EnumOption{Allowed: map[string]bool{"none": true, "strict": true, "loose": true}},
		"sec":                 options.EnumOption{Allowed: map[string]bool{"none": true, "krb5": true, "krb5i": true, "ntlm": true, "ntlmi": true, "ntlmv2": true, "ntlmv2i": true, "ntlmssp": true, "ntlmsspi": true}},
		"upcall_target":       options.EnumOption{Allowed: map[string]bool{"mount": true, "app": true}},
		"vers":                options.EnumOption{Allowed: map[string]bool{"1.0": true, "2.0": true, "2.1": true, "3.0": true, "3.02": true, "3.0.2": true, "3.1.1": true, "3.11": true, "3": true, "default": true}},
		"forceuid":            options.NoValueOption{},
		"forcegid":            options.NoValueOption{},
		"idsfromsid":          options.NoValueOption{},
		"domainauto":          options.NoValueOption{},
		"guest":               options.NoValueOption{},
		"setuids":             options.NoValueOption{},
		"nosetuids":           options.NoValueOption{},
		"perm":                options.NoValueOption{},
		"noperm":              options.NoValueOption{},
		"dynperm":             options.NoValueOption{},
		"nostrictsync":        options.NoValueOption{},
		"handlecache":         options.NoValueOption{},
		"nohandlecache":       options.NoValueOption{},
		"rwpidforward":        options.NoValueOption{},
		"mapchars":            options.NoValueOption{},
		"nomapchars":          options.NoValueOption{},
		"mapposix":            options.NoValueOption{},
		"intr":                options.NoValueOption{},
		"nointr":              options.NoValueOption{},
		"hard":                options.NoValueOption{},
		"soft":                options.NoValueOption{},
		"noacl":               options.NoValueOption{},
		"cifsacl":             options.NoValueOption{},
		"nocase":              options.NoValueOption{},
		"ignorecase":          options.NoValueOption{},
		"seal":                options.NoValueOption{},
		"rdma":                options.NoValueOption{},
		"resilienthandles":    options.NoValueOption{},
		"noresilienthandles":  options.NoValueOption{},
		"persistenthandles":   options.NoValueOption{},
		"nopersistenthandles": options.NoValueOption{},
		"nobrl":               options.NoValueOption{},
		"forcemandatorylock":  options.NoValueOption{},
		"locallease":          options.NoValueOption{},
		"nolease":             options.NoValueOption{},
		"sfu":                 options.NoValueOption{},
		"mfsymlinks":          options.NoValueOption{},
		"serverino":           options.NoValueOption{},
		"noserverino":         options.NoValueOption{},
		"posix":               options.NoValueOption{},
		"unix":                options.NoValueOption{},
		"linux":               options.NoValueOption{},
		"noposix":             options.NoValueOption{},
		"nounix":              options.NoValueOption{},
		"nolinux":             options.NoValueOption{},
		"nouser_xattr":        options.NoValueOption{},
		"nodfs":               options.NoValueOption{},
		"noautotune":          options.NoValueOption{},
		"nosharesock":         options.NoValueOption{},
		"noblocksend":         options.NoValueOption{},
		"fsc":                 options.NoValueOption{},
		"multiuser":           options.NoValueOption{},
		"noposixpaths":        options.NoValueOption{},
		"posixpaths":          options.NoValueOption{},
		"compress":            options.NoValueOption{},
		"sloppy":              options.NoValueOption{},
		"max_cached_dirs":     options.IntOption{},
		"handletimeout":       options.IntOption{},
		"echo_interval":       options.IntOption{},
		"rsize":               options.SizeOption{},
		"wsize":               options.SizeOption{},
		"bsize":               options.SizeOption{},
		"max_credits":         options.IntOption{},
		"actimeo":             options.IntOption{},
		"acregmax":            options.IntOption{},
		"acdirmax":            options.IntOption{},
		"max_channels":        options.IntOption{},
		"closetimeo":          options.IntOption{},
		"esize":               options.SizeOption{},
	}

	mergedOptions := options.MergeOptions(options.CommonOptions(), fsOptions)

	return options.ValidateOptionsMap(ctx, api, entry, "cifs", mergedOptions), nil
}
