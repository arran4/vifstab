package options

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
)

// UmaskOption validates octal umask permissions (e.g. 022, 077, 0022).
type UmaskOption struct {
	CustomErrFormat string
}

func (u UmaskOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return (OctalOption{CustomErrFormat: u.CustomErrFormat}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// ModeOption validates octal mode permissions (e.g. 0755, 0644).
type ModeOption struct {
	CustomErrFormat string
}

func (m ModeOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return (OctalOption{CustomErrFormat: m.CustomErrFormat}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// FileModeOption validates file octal mode permissions.
type FileModeOption struct {
	CustomErrFormat string
}

func (f FileModeOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return (OctalOption{CustomErrFormat: f.CustomErrFormat}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// DirModeOption validates directory octal mode permissions.
type DirModeOption struct {
	CustomErrFormat string
}

func (d DirModeOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return (OctalOption{CustomErrFormat: d.CustomErrFormat}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}
