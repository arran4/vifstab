package options

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// SystemdVersionKey holds an explicit target-version observation in the existing
// rules cache. No version is inferred from PID1, distro releases or a binary name.
const SystemdVersionKey rules.CacheKey = "systemd_version"

// TargetSystemdVersion describes a supplied target, which need not be this host.
// Major <= 0 means unknown. Providers may also supply a decimal string (fixtures).
type TargetSystemdVersion struct{ Major int }

type systemdShape uint8

const (
	systemdFlag systemdShape = iota
	systemdDuration
	systemdDependency
	systemdAbsolutePath
	systemdBoolean
	systemdGraceful
)

type systemdSpec struct {
	shape      systemdShape
	minimum    int
	repeatable bool
}

// Catalogue facts are checked against upstream ec1065495802b32b42728914c29cc7252a9910fb
// and historical release manuals. This is an offline catalogue, not a claim that
// a generator, kernel, filesystem or initrd context will implement each option.
var systemdCatalogue = map[string]systemdSpec{
	"x-systemd.automount":           {systemdFlag, 215, false},
	"x-systemd.makefs":              {systemdFlag, 236, false},
	"x-systemd.growfs":              {systemdFlag, 236, false},
	"x-systemd.rw-only":             {systemdFlag, 246, false},
	"x-systemd.pcrfs":               {systemdFlag, 253, false},
	"x-systemd.validatefs":          {systemdFlag, 258, false},
	"x-systemd.idle-timeout":        {systemdDuration, 220, false},
	"x-systemd.device-timeout":      {systemdDuration, 215, false},
	"x-systemd.mount-timeout":       {systemdDuration, 233, false},
	"x-systemd.requires":            {systemdDependency, 220, true},
	"x-systemd.wants":               {systemdDependency, 257, true},
	"x-systemd.before":              {systemdDependency, 233, true},
	"x-systemd.after":               {systemdDependency, 233, true},
	"x-systemd.wanted-by":           {systemdDependency, 245, true},
	"x-systemd.required-by":         {systemdDependency, 245, true},
	"x-systemd.onfailure":           {systemdDependency, 262, true},
	"x-systemd.requires-mounts-for": {systemdAbsolutePath, 220, true},
	"x-systemd.wants-mounts-for":    {systemdAbsolutePath, 256, true},
	"x-systemd.device-bound":        {systemdBoolean, 233, false},
	"x-systemd.graceful-option":     {systemdGraceful, 258, true},
}

func targetSystemdVersion(ctx context.Context) int {
	cache := rules.CacheFromContext(ctx)
	if cache == nil {
		return 0
	}
	value, _ := cache.Get(SystemdVersionKey)
	version := 0
	switch v := value.(type) {
	case TargetSystemdVersion:
		version = v.Major
	case string:
		if v != "" && strings.Trim(v, "0123456789") == "" {
			version, _ = strconv.Atoi(v)
		}
	}
	if version < 0 {
		return 0
	}
	return version
}

// ValidateSystemdOptions validates generator options independently of any native
// filesystem option map, including entries with swap/bind or unknown fs types.
// It does not require systemd to be installed and performs no provider calls.
func ValidateSystemdOptions(ctx context.Context, entry *rules.FstabEntry) []rules.ValidationIssue {
	if len(entry.Fields) < 4 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") || entry.Fields[2] == "ignore" {
		return nil
	}
	set := ParseOptionSet(entry.Fields[3])
	var issues []rules.ValidationIssue
	for index, state := range set.ordered {
		if strings.HasPrefix(state.Key, "x-systemd.") {
			issues = append(issues, validateSystemdOption(ctx, entry, state.Key, state.Value, state.HasVal, set, index)...)
		}
	}
	return issues
}

func validateSystemdOption(ctx context.Context, entry *rules.FstabEntry, key, value string, hasValue bool, set OptionSet, index int) []rules.ValidationIssue {
	issue := func(level rules.IssueLevel, reason string) rules.ValidationIssue {
		return rules.ValidationIssue{Message: fmt.Sprintf("systemd option '%s' on line %d: %s", key, entry.LineNum, reason), Line: entry.RawLine, Level: level}
	}
	spec, known := systemdCatalogue[key]
	if !known {
		return []rules.ValidationIssue{{Message: fmt.Sprintf("Unverified systemd extension '%s' on line %d: value syntax and semantics were not checked", key, entry.LineNum), Line: entry.RawLine, Level: rules.IssueLevelInfo}}
	}
	var issues []rules.ValidationIssue
	decoded := decodeFstabOptionValue(value)
	if spec.shape == systemdFlag {
		if hasValue {
			issues = append(issues, issue(rules.IssueLevelError, "takes no value; remove the '=' and value"))
		}
	} else if spec.shape != systemdBoolean && (!hasValue || decoded == "") {
		issues = append(issues, issue(rules.IssueLevelError, "requires a nonempty value after '='"))
	} else {
		reason, unverified := validateSystemdValue(spec.shape, decoded, hasValue)
		if reason != "" {
			level := rules.IssueLevelError
			if unverified {
				level = rules.IssueLevelInfo
			}
			issues = append(issues, issue(level, reason))
		}
	}
	first, _ := set.FirstIndex(key)
	if first != index {
		return issues
	}
	if !spec.repeatable && len(set.byKey[key]) > 1 {
		issues = append(issues, issue(rules.IssueLevelInfo, "repeated singleton setting; all occurrences were checked, but review the repeated values and intended override"))
	}
	version := targetSystemdVersion(ctx)
	if version > 0 && version < spec.minimum {
		issues = append(issues, issue(rules.IssueLevelWarning, fmt.Sprintf("requires systemd %d or newer; the supplied target version is %d", spec.minimum, version)))
	} else if version == 0 && firstKnownSystemdIndex(set) == index {
		issues = append(issues, issue(rules.IssueLevelInfo, "target systemd version/applicability unknown; syntax checks do not establish generator behavior"))
	}
	automount := hasBareOption(set, "x-systemd.automount") || hasOptionValue(set, "comment", "systemd.automount")
	switch key {
	case "x-systemd.automount":
		if automount && (set.Has("noauto") || set.Has("auto")) {
			issues = append(issues, issue(rules.IssueLevelInfo, "auto/noauto have no effect when systemd automount is requested; the automount unit controls on-demand activation"))
		}
		if len(entry.Fields) > 2 && entry.Fields[2] == "swap" {
			issues = append(issues, issue(rules.IssueLevelInfo, "automount does not apply to swap entries"))
		}
	case "x-systemd.idle-timeout":
		if !automount {
			issues = append(issues, issue(rules.IssueLevelInfo, "idle timeout is for an automount unit; add x-systemd.automount or remove the ineffective timeout"))
		}
	case "x-systemd.makefs":
		issues = append(issues, issue(rules.IssueLevelInfo, "requests device initialization when systemd considers it empty; review this potentially destructive side effect (contents are not inspected by vifstab)"))
	case "x-systemd.device-timeout":
		if len(entry.Fields) > 0 && !strings.HasPrefix(rules.ResolveDevicePath(entry.Fields[0]), "/dev/") {
			issues = append(issues, issue(rules.IssueLevelInfo, "device timeout applies to a device-backed entry; this source is not a /dev/ reference after tag translation"))
		}
	}
	return issues
}

func firstKnownSystemdIndex(set OptionSet) int {
	first := len(set.ordered)
	for key := range systemdCatalogue {
		if i, ok := set.FirstIndex(key); ok && i < first {
			first = i
		}
	}
	return first
}

func hasBareOption(set OptionSet, key string) bool {
	for _, state := range set.byKey[key] {
		if !state.HasVal {
			return true
		}
	}
	return false
}

func hasOptionValue(set OptionSet, key, value string) bool {
	for _, state := range set.byKey[key] {
		if state.HasVal && state.Value == value {
			return true
		}
	}
	return false
}

func decodeFstabOptionValue(value string) string {
	// Decode the standard fstab field escapes without changing saved spelling.
	// A single pass matters: \134040 denotes a literal backslash followed by 040.
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+3 < len(value) {
			var decoded byte
			switch value[i+1 : i+4] {
			case "040":
				decoded = ' '
			case "011":
				decoded = '\t'
			case "012":
				decoded = '\n'
			case "134":
				decoded = '\\'
			}
			if decoded != 0 {
				b.WriteByte(decoded)
				i += 3
				continue
			}
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func validateSystemdValue(shape systemdShape, value string, hasValue bool) (reason string, unverified bool) {
	if strings.ContainsRune(value, 0) {
		return "NUL is not valid in a generator option value", false
	}
	switch shape {
	case systemdDuration:
		if strings.ContainsAny(value, "\\\"") {
			return "quoted or remaining escaped duration syntax is unverified; use a documented unquoted time span", true
		}
		if !validSystemdDuration(value) {
			return "invalid nonnegative systemd time span; use e.g. 30s, 2min, 1h30min or infinity", false
		}
	case systemdBoolean:
		if hasValue {
			switch strings.ToLower(value) {
			case "1", "yes", "y", "true", "t", "on", "0", "no", "n", "false", "f", "off":
			default:
				return "expected a supported boolean (e.g. yes/no) or a bare option; an empty '=' is not a boolean", false
			}
		}
	case systemdAbsolutePath:
		if !strings.HasPrefix(value, "/") {
			return "requires an absolute path; use a path beginning with '/'", false
		}
	case systemdDependency:
		if strings.HasPrefix(value, "/") || validSystemdUnitName(value) {
			return "", false
		}
		// Upstream also mangles shorthand/other strings. Do not reject that
		// supported behavior by imposing an invented closed unit-name grammar.
		if len(value) > 255 {
			return "unit name is too long; use a valid unit name of at most 255 bytes or an absolute path", false
		}
		return "dependency spelling needs generator mangling and is unverified; prefer an explicit unit name or absolute device/mount path", true
	case systemdGraceful:
		name, _, _ := strings.Cut(value, "=")
		if name == "" || strings.ContainsAny(name, " \t\r\n,") {
			return "requires a single nonempty kernel mount-option name, optionally followed by '=value'", false
		}
		if strings.HasPrefix(name, "x-") || strings.HasPrefix(name, "X-") {
			return "requires a kernel mount option, not another userspace extension", false
		}
		switch name {
		case "auto", "noauto", "nofail", "fail", "user", "nouser", "users", "owner", "group", "_netdev", "defaults", "comment":
			return "requires a kernel mount option; this name is a userspace control option", false
		}
	}
	return "", false
}

func validSystemdUnitName(value string) bool {
	if len(value) == 0 || len(value) > 255 || value[0] == '@' {
		return false
	}
	i := strings.LastIndexByte(value, '.')
	if i <= 0 {
		return false
	}
	switch value[i+1:] {
	case "service", "socket", "device", "mount", "automount", "swap", "target", "path", "timer", "slice", "scope":
	default:
		return false
	}
	for _, c := range value[:i] {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune(":-_.\\@", c) {
			continue
		}
		return false
	}
	return true
}
