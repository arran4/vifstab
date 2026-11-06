package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// GenericExtensionOptions is the small catalogue of util-linux userspace
// options whose value shapes we validate. It is not a closed namespace.
// See util-linux sys-utils/mount.8.adoc at bd3a77c82ba7d88235e2d8942b248e0f69488f2e.
func GenericExtensionOptions() map[string]OptionValidator {
	return map[string]OptionValidator{
		"X-mount.mkdir":  OptionalOctalOption{},
		"x-mount.mkdir":  OptionalOctalOption{}, // Documented, deprecated alias.
		"X-mount.noloop": NoValueOption{},
		"X-mount.subdir": StringOption{}, // Syntax only; never inspect contents.
	}
}

// ValidateExtensionOption shares namespace policy with older filesystem rules
// which retain custom per-option loops. handled is false for ordinary options.
// ValidateOptionsMap gives explicit map validators precedence. Older loops have
// no filesystem-specific extension policy and delegate extension tokens here.
func ValidateExtensionOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName, key, value string, hasValue bool) (issues []rules.ValidationIssue, handled bool) {
	namespace := extensionNamespace(key)
	if namespace == "" {
		return nil, false
	}
	if validator, ok := GenericExtensionOptions()[key]; ok {
		return validator.ValidateOption(ctx, api, entry, fsName, key, value, hasValue, nil), true
	}
	return []rules.ValidationIssue{{
		Message: fmt.Sprintf("Unverified %s extension '%s' on line %d: value syntax and semantics were not checked", namespace, key, entry.LineNum),
		Line:    entry.RawLine,
		Level:   rules.IssueLevelInfo,
	}}, true
}

// extensionNamespace keeps forward-compatibility policy in the shared entry
// point. Case matters: X-* is not a blanket alias for x-* or x-systemd.*.
func extensionNamespace(key string) string {
	switch {
	case strings.HasPrefix(key, "x-systemd."):
		return "systemd"
	case strings.HasPrefix(key, "x-"):
		return "userspace x-"
	case strings.HasPrefix(key, "X-"):
		return "userspace X-"
	default:
		return ""
	}
}
