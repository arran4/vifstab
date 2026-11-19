package options_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type accountProbeAPI struct {
	rules.API
	calls int
}

func (a *accountProbeAPI) CheckUserExists(int) rules.ExistState  { a.calls++; return rules.Exists }
func (a *accountProbeAPI) CheckGroupExists(int) rules.ExistState { a.calls++; return rules.Exists }

func TestNumericAccountIDRangeDoesNotDependOnLookup(t *testing.T) {
	entry := &rules.FstabEntry{LineNum: 1, RawLine: "fixture"}
	for _, validator := range []options.OptionValidator{options.UIDOption{}, options.GIDOption{}, options.UIDOrEnumOption{}} {
		for _, value := range []string{"-1", "4294967295", "4294967296"} {
			api := &accountProbeAPI{}
			issues := validator.ValidateOption(context.Background(), api, entry, "fixturefs", "id", value, true, nil)
			if api.calls != 0 || len(issues) != 1 || issues[0].Level != rules.IssueLevelError {
				t.Fatalf("invalid ID %s depended on observation: calls=%d issues=%v", value, api.calls, issues)
			}
		}
		for _, value := range []string{"0", "+1000", "001000", "4294967294"} {
			api := &accountProbeAPI{}
			issues := validator.ValidateOption(context.Background(), api, entry, "fixturefs", "id", value, true, nil)
			if strconv.IntSize == 32 && value == "4294967294" {
				if api.calls != 0 || len(issues) != 1 || issues[0].Level != rules.IssueLevelInfo {
					t.Fatalf("unrepresentable valid ID was wrapped or rejected: %v", issues)
				}
			} else if api.calls != 1 || len(issues) != 0 {
				t.Fatalf("valid ID %s lost syntax support: calls=%d issues=%v", value, api.calls, issues)
			}
		}
	}
	api := &accountProbeAPI{}
	validator := options.UIDOrEnumOption{AllowedEnum: map[string]bool{"-1": true}}
	if issues := validator.ValidateOption(context.Background(), api, entry, "fixturefs", "id", "-1", true, nil); len(issues) != 0 || api.calls != 0 {
		t.Fatalf("explicit filesystem enum was reinterpreted as an ID: %v", issues)
	}
}
