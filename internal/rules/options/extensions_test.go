package options_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestExtensionPolicy(t *testing.T) {
	cases := []struct {
		option string
		level  rules.IssueLevel
		text   string
	}{
		{option: "X-mount.mkdir"},
		{option: "X-mount.mkdir=0755"},
		{option: "x-mount.mkdir=0700"},
		{option: "X-mount.noloop"},
		{option: "X-mount.subdir=path"},
		{option: "X-mount.subdir=a=b"},
		{option: "X-mount.mkdir=not-octal", level: rules.IssueLevelError, text: "must be an octal integer"},
		{option: "x-mount.mkdir=0789", level: rules.IssueLevelError, text: "must be an octal integer"},
		{option: "X-mount.mkdir=", level: rules.IssueLevelError, text: "Missing value"},
		{option: "x-mount.mkdir=", level: rules.IssueLevelError, text: "Missing value"},
		{option: "X-mount.noloop=", level: rules.IssueLevelError, text: "does not take a value"},
		{option: "X-mount.noloop=yes", level: rules.IssueLevelError, text: "does not take a value"},
		{option: "X-mount.subdir", level: rules.IssueLevelError, text: "Missing value"},
		{option: "X-mount.subdir=", level: rules.IssueLevelError, text: "Missing value"},
		{option: "x-future.option=anything", level: rules.IssueLevelInfo, text: "Unverified userspace x- extension"},
		{option: "X-future.option", level: rules.IssueLevelInfo, text: "Unverified userspace X- extension"},
		{option: "x-mount.noloop=value", level: rules.IssueLevelInfo, text: "Unverified userspace x- extension"},
		{option: "X-systemd.automount=no", level: rules.IssueLevelInfo, text: "Unverified userspace X- extension"},
		{option: "x-systemd.future=value", level: rules.IssueLevelInfo, text: "Unverified systemd extension"},
		{option: "x-systemd.automount=no", level: rules.IssueLevelError, text: "takes no value"},
		{option: "x-mount.mkidr=0755", level: rules.IssueLevelInfo, text: "Unverified"},
		{option: "ordinary-unknown=value", level: rules.IssueLevelError, text: "Unknown"},
	}
	for _, fsName := range []string{"ext4", "vfat", "tmpfs", "nfs"} {
		for _, tc := range cases {
			t.Run(fsName+"/"+tc.option, func(t *testing.T) {
				entry := &rules.FstabEntry{LineNum: 7, RawLine: "example /example " + fsName + " " + tc.option + " 0 0", Fields: []string{"example", "/example", fsName, tc.option, "0", "0"}}
				// Generic userspace extensions are checked even for a filesystem
				// whose map does not include CommonOptions. No API calls needed.
				issues := append(options.ValidateOptionsMap(systemdContext("262"), nil, entry, fsName, nil), options.ValidateSystemdOptions(systemdContext("262"), entry)...)
				if tc.level == "" {
					if len(issues) != 0 {
						t.Fatalf("valid option: %v", issues)
					}
					return
				}
				if len(issues) != 1 || issues[0].Level != tc.level || !strings.Contains(issues[0].Message, tc.text) || !strings.Contains(issues[0].Message, "line 7") || issues[0].Line != entry.RawLine {
					t.Fatalf("want %s %q, got %v", tc.level, tc.text, issues)
				}
				if tc.level == rules.IssueLevelInfo && rules.IsBlockingIssue(issues[0]) {
					t.Fatal("future extension must remain nonblocking")
				}
			})
		}
	}
}

func TestRegisteredSystemdValidatorPrecedesNamespacePolicy(t *testing.T) {
	entry := &rules.FstabEntry{LineNum: 3, Fields: []string{"test", "/test", "test", "x-systemd.automount=no,x-systemd.future=new,X-systemd.automount"}}
	optionMap := map[string]options.OptionValidator{"x-systemd.automount": options.NoValueOption{}}
	issues := options.ValidateOptionsMap(context.Background(), nil, entry, "test", optionMap)
	if len(issues) != 2 || issues[0].Level != rules.IssueLevelError || issues[1].Level != rules.IssueLevelInfo {
		t.Fatalf("dedicated validator was bypassed or namespaces conflated: %v", issues)
	}
}
