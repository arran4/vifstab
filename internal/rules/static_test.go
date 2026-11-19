package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/rules"
)

func TestStaticObservationsRemainUnknown(t *testing.T) {
	api := rules.StaticAPI{}
	for _, state := range []rules.ExistState{api.CheckFileExists("/missing"), api.CheckBlockDeviceExists("UUID=missing"), api.CheckUserExists(1234), api.CheckGroupExists(2345)} {
		if state != rules.Unknown {
			t.Fatalf("static observation became %v", state)
		}
	}
	inv, err := api.GetBlockDevices()
	if inv.Available || !errors.Is(err, rules.ErrStaticObservation) {
		t.Fatalf("inventory: %+v, %v", inv, err)
	}
	if fs, err := api.GetSupportedFilesystems(); fs != nil || !errors.Is(err, rules.ErrStaticObservation) {
		t.Fatalf("filesystems: %v,%v", fs, err)
	}
	if api.GetHostKernelVersion() != "" {
		t.Fatal("invented kernel version")
	}
	if got := api.ResolveDevicePath("UUID=fixture"); got != rules.ResolveDevicePath("UUID=fixture") {
		t.Fatalf("resolver differs: %s", got)
	}
}

func TestStaticValidationKeepsSyntaxAndUnknownSeparate(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		blocked    bool
	}{
		{"unknown host", "UUID=fixture /fixture ext4 defaults,data=journal 0 2\n", false},
		{"invalid syntax", "UUID=fixture /fixture ext4 definitely-invalid-option 0 2\n", true},
		{"malformed entry", "bad\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := editor.VerifyAndLintReader(strings.NewReader(tc.text), rules.StaticAPI{})
			if err != nil {
				t.Fatal(err)
			}
			blocked := false
			for _, issue := range issues {
				blocked = blocked || rules.IsBlockingIssue(issue)
				if strings.Contains(issue.Message, "does not exist") || strings.Contains(issue.Message, "deprecated since kernel") {
					t.Fatalf("unknown host observation became certainty: %+v", issue)
				}
			}
			if blocked != tc.blocked {
				t.Fatalf("blocking=%v, issues=%+v", blocked, issues)
			}
		})
	}
}
