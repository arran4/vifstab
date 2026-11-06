package options_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func systemdContext(value any) context.Context {
	cache := rules.NewCache()
	cache.Set(options.SystemdVersionKey, value)
	return rules.WithCache(context.Background(), cache)
}

func systemdIssues(ctx context.Context, raw string) []rules.ValidationIssue {
	entry := &rules.FstabEntry{LineNum: 9, RawLine: "UUID=example /target ext4 " + raw + " 0 2", Fields: []string{"UUID=example", "/target", "ext4", raw, "0", "2"}}
	// A nil provider proves these checks do not need discovery, a daemon or VFS.
	return options.ValidateSystemdOptions(ctx, entry)
}

func errorCount(issues []rules.ValidationIssue) int {
	n := 0
	for _, issue := range issues {
		if issue.Level == rules.IssueLevelError {
			n++
		}
	}
	return n
}

func matchingIssues(issues []rules.ValidationIssue, text string) int {
	n := 0
	for _, issue := range issues {
		if strings.Contains(issue.Message, text) {
			n++
		}
	}
	return n
}

func TestSystemdKnownSyntax(t *testing.T) {
	ctx := systemdContext(options.TargetSystemdVersion{Major: 262})
	for _, flag := range []string{"automount", "makefs", "growfs", "rw-only", "pcrfs", "validatefs"} {
		for _, suffix := range []string{"", "=", "=no", "=yes=extra"} {
			t.Run(flag+suffix, func(t *testing.T) {
				want := 0
				if suffix != "" {
					want = 1
				}
				issues := systemdIssues(ctx, "x-systemd."+flag+suffix)
				if errorCount(issues) != want {
					t.Fatalf("flag shape: %v", issues)
				}
			})
		}
	}
	valid := []string{
		"idle-timeout=2min", "mount-timeout=infinity", "device-timeout=0", `mount-timeout=1h\04030min`,
		"requires=first.mount", "wants=/dev/disk/by-label/example", "before=local-fs.target", "after=/other/mount",
		"wanted-by=multi-user.target", "required-by=remote-fs.target", "onfailure=rescue.target",
		"requires-mounts-for=/absolute/path", "wants-mounts-for=/other", "device-bound", "device-bound=YeS", "device-bound=f",
		"graceful-option=nosymfollow", "graceful-option=some_kernel_option=value=extra",
	}
	for _, value := range valid {
		if issues := systemdIssues(ctx, "x-systemd."+value); errorCount(issues) != 0 {
			t.Errorf("valid %s: %v", value, issues)
		}
	}
	invalid := []string{
		"idle-timeout", "idle-timeout=", "idle-timeout=garbage", "idle-timeout=-0", "mount-timeout=30s=extra", "device-timeout=never",
		"requires", "requires=", "wants=", "before=", "after=", "wanted-by=", "required-by=", "onfailure=",
		"requires-mounts-for=relative", "wants-mounts-for=", "device-bound=", "device-bound=maybe",
		"graceful-option", "graceful-option=", "graceful-option==value", "graceful-option=x-systemd.automount", "graceful-option=_netdev",
	}
	for _, value := range invalid {
		issues := systemdIssues(ctx, "x-systemd."+value)
		if errorCount(issues) != 1 || !strings.Contains(issues[0].Message, "line 9") || issues[0].Line == "" {
			t.Errorf("invalid %s: %v", value, issues)
		}
	}
}

func TestSystemdRepeatabilityAndInteractions(t *testing.T) {
	ctx := systemdContext("262")
	cases := []struct {
		raw    string
		errors int
		text   string
		count  int
	}{
		{"x-systemd.requires=first.mount,x-systemd.requires=second.mount,x-systemd.requires=third.mount", 0, "singleton", 0},
		{"x-systemd.requires=,x-systemd.requires=second.mount", 1, "requires a nonempty", 1},
		{"x-systemd.requires=first.mount,x-systemd.requires=", 1, "requires a nonempty", 1},
		{"x-systemd.mount-timeout=30s,x-systemd.mount-timeout=2min", 0, "repeated singleton", 1},
		{"x-systemd.mount-timeout=30s,x-systemd.mount-timeout=30s", 0, "repeated singleton", 1},
		{"x-systemd.mount-timeout=bad,x-systemd.mount-timeout=30s", 1, "repeated singleton", 1},
		{"x-systemd.graceful-option=first,x-systemd.graceful-option=second", 0, "singleton", 0},
		{"noauto,x-systemd.automount,x-systemd.idle-timeout=30s", 0, "auto/noauto have no effect", 1},
		{"auto,x-systemd.automount", 0, "auto/noauto have no effect", 1},
		{"x-systemd.idle-timeout=30s", 0, "idle timeout is for an automount unit", 1},
		{"comment=systemd.automount,x-systemd.idle-timeout=30s", 0, "idle timeout is for an automount unit", 0},
		{"x-systemd.makefs", 0, "potentially destructive", 1},
		{"x-systemd.requires=shorthand", 0, "mangling", 1},
		{`x-systemd.mount-timeout="30s"`, 0, "unverified", 1},
		{"x-systemd.future=anything", 0, "Unverified systemd extension", 1},
		{"X-systemd.automount=no", 0, "systemd", 0},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			issues := systemdIssues(ctx, tc.raw)
			if errorCount(issues) != tc.errors || matchingIssues(issues, tc.text) != tc.count {
				t.Fatalf("errors want%d, %q want%d: %v", tc.errors, tc.text, tc.count, issues)
			}
			for _, issue := range issues {
				if issue.Level == rules.IssueLevelInfo && rules.IsBlockingIssue(issue) {
					t.Fatal("context/redundancy/unknown info became blocking")
				}
			}
		})
	}
}

func TestSystemdVersionObservations(t *testing.T) {
	for _, observation := range []any{nil, "", "garbage", "258-custom", "-1", true, options.TargetSystemdVersion{}, options.TargetSystemdVersion{Major: -1}} {
		issues := systemdIssues(systemdContext(observation), "x-systemd.automount,x-systemd.mount-timeout=30s")
		if len(issues) != 1 || issues[0].Level != rules.IssueLevelInfo || matchingIssues(issues, "unknown") != 1 || matchingIssues(issues, "supplied target") != 0 {
			t.Errorf("unknown observation %#v: %v", observation, issues)
		}
	}
	// Compare documented introduction boundaries, including independently checked
	// v255/v256 wants-mounts-for and v256/v257 wants, v257/v258 graceful/validatefs.
	for _, tc := range []struct {
		raw string
		min int
	}{
		{"x-systemd.automount", 215}, {"x-systemd.requires=other.mount", 220}, {"x-systemd.before=other.mount", 233},
		{"x-systemd.makefs", 236}, {"x-systemd.wanted-by=other.target", 245}, {"x-systemd.rw-only", 246},
		{"x-systemd.pcrfs", 253}, {"x-systemd.wants-mounts-for=/other", 256}, {"x-systemd.wants=other.mount", 257},
		{"x-systemd.graceful-option=nosymfollow", 258}, {"x-systemd.validatefs", 258}, {"x-systemd.onfailure=rescue.target", 262},
	} {
		for _, version := range []int{tc.min - 1, tc.min, 999} {
			issues := systemdIssues(systemdContext(options.TargetSystemdVersion{Major: version}), tc.raw)
			warnings := 0
			for _, issue := range issues {
				if issue.Level == rules.IssueLevelWarning {
					warnings++
					if !strings.Contains(issue.Message, "supplied target version") {
						t.Errorf("invented host-version claim: %v", issue)
					}
				}
			}
			want := 0
			if version < tc.min {
				want = 1
			}
			if warnings != want || matchingIssues(issues, "version/applicability unknown") != 0 {
				t.Errorf("%s target%d: %v", tc.raw, version, issues)
			}
		}
	}
}

func TestSystemdDiagnosticOrderAndSourcePreservation(t *testing.T) {
	raw := "x-systemd.mount-timeout=bad,ordinary-unknown,x-systemd.idle-timeout=bad"
	issues := systemdIssues(systemdContext("262"), raw)
	if len(issues) != 3 || !strings.Contains(issues[0].Message, "mount-timeout") || !strings.Contains(issues[1].Message, "idle-timeout") {
		t.Fatalf("source order changed: %v", issues)
	}
	set := options.ParseOptionSet(raw)
	if i, ok := set.FirstIndex("x-systemd.idle-timeout"); !ok || i != 2 {
		t.Fatalf("first index=%d %t", i, ok)
	}
	var empty options.OptionSet
	if _, ok := empty.FirstIndex("missing"); ok {
		t.Fatal("empty set invented a first index")
	}
}
