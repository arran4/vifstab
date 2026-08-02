package ecryptfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type EcryptfsRule struct{}

func init() {
	rules.Register(&EcryptfsRule{})
}

func (r *EcryptfsRule) Name() string {
	return "ecryptfs"
}

func (r *EcryptfsRule) Dependencies() []string {
	return []string{}
}

func (r *EcryptfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ecryptfs" {
		return nil, nil
	}

	issues := options.ValidateOptionsMap(ctx, api, entry, "ecryptfs", ecryptfsOptions)

	// Validate options embedded in the "key" parameter (e.g. key=passphrase:passphrase_passwd_file=/mnt/file.txt)
	parsedOpts := rules.SplitOptions(entry.Fields[3])
	for _, opt := range parsedOpts {
		if strings.HasPrefix(opt, "key=") {
			val := strings.TrimPrefix(opt, "key=")
			parts := strings.Split(val, ":")
			if len(parts) > 1 {
				// The parts[1:] might contain options similar to the top-level options.
				for _, subOpt := range parts[1:] {
					subOptParts := strings.SplitN(subOpt, "=", 2)
					subKey := subOptParts[0]
					subVal := ""
					hasSubVal := false
					if len(subOptParts) == 2 {
						subVal = subOptParts[1]
						hasSubVal = true
					}

					// Re-use ecryptfsOptions to validate embedded options
					if validator, exists := ecryptfsOptions[subKey]; exists {
						subIssues := validator.ValidateOption(ctx, api, entry, "ecryptfs key sub-option", subKey, subVal, hasSubVal, nil)
						issues = append(issues, subIssues...)
					}
				}
			}
		}
	}

	return issues, nil
}

var ecryptfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"ecryptfs_sig":                    options.StringOption{},
	"ecryptfs_fnek_sig":               options.StringOption{},
	"ecryptfs_cipher":                 options.StringOption{},
	"ecryptfs_key_bytes":              options.IntOption{},
	"ecryptfs_passthrough":            options.NoValueOption{},
	"no_sig_cache":                    options.NoValueOption{},
	"ecryptfs_encrypted_view":         options.NoValueOption{},
	"ecryptfs_xattr":                  options.NoValueOption{},
	"verbose":                         options.NoValueOption{},
	"key":                             options.StringOption{}, // Allows sub-options separated by :
	"ecryptfs_enable_filename_crypto": options.EnumOption{Allowed: map[string]bool{"y": true, "n": true}},
	"verbosity":                       options.EnumOption{Allowed: map[string]bool{"0": true, "1": true}},
	"passphrase_passwd":               options.StringOption{},
	"passphrase_passwd_file":          options.FileOption{CheckExistence: true},
	"passphrase_passwd_fd":            options.IntOption{},
	"passphrase_salt":                 options.StringOption{},
	"openssl_keyfile":                 options.FileOption{CheckExistence: true},
	"openssl_passwd_file":             options.FileOption{CheckExistence: true},
	"openssl_passwd_fd":               options.IntOption{},
	"openssl_passwd":                  options.StringOption{},
})
