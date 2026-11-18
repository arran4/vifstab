package options_test

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/format"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/core"
	"github.com/arran4/vifstab/internal/rules/options"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/ordered-*.txtar
var orderedFixtures embed.FS

// These checks exercise the shared API; they do not register systemd policy.
// Dependency keys collect every value, while the example singleton check sees
// both differing and identical repetitions and emits exactly one finding.
func TestOrderedOptionsTXTAR(t *testing.T) {
	files, err := orderedFixtures.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			data, err := orderedFixtures.ReadFile("testdata/" + file.Name())
			if err != nil {
				t.Fatal(err)
			}
			sections := make(map[string]string)
			for _, section := range txtar.Parse(data).Files {
				sections[section.Name] = string(section.Data)
			}
			input := sections["input/etc/fstab"]
			entries, err := core.ParseFstab(strings.NewReader(input))
			if err != nil || len(entries) != 1 {
				t.Fatalf("parse: entries=%v error=%v", entries, err)
			}
			entry := entries[0]
			originalFields := append([]string(nil), entry.Fields...)
			originalRaw := entry.RawLine
			var observed []options.OptionState
			var invocationOrder []string
			legacyCalls := 0
			legacy := options.OptionValidatorFunc(func(_ context.Context, _ rules.API, _ *rules.FstabEntry, _, key, value string, hasValue bool, last map[string]options.OptionState) []rules.ValidationIssue {
				legacyCalls++
				want, exists := options.ParseOptionSet(originalFields[3]).Last(key)
				if !exists || last[key] != want {
					t.Errorf("legacy view lost last value for %s", key)
				}
				return nil
			})
			check := options.OptionSetValidatorFunc(func(_ context.Context, _ rules.API, entry *rules.FstabEntry, _, key string, set options.OptionSet) []rules.ValidationIssue {
				invocationOrder = append(invocationOrder, key)
				if observed == nil {
					observed = set.Ordered()
				} else if !reflect.DeepEqual(observed, set.Ordered()) {
					t.Error("different checks observed different sequences")
				}
				if key == "x-systemd.idle-timeout" && len(set.All(key)) > 1 {
					return []rules.ValidationIssue{{Message: fmt.Sprintf("example singleton %s has %d occurrences", key, len(set.All(key))), Line: entry.RawLine, Level: rules.IssueLevelInfo}}
				}
				return nil
			})
			optionMap := make(map[string]options.OptionValidator)
			var expectedKeys []string
			for _, state := range options.ParseOptionSet(entry.Fields[3]).Ordered() {
				if _, exists := optionMap[state.Key]; !exists {
					expectedKeys = append(expectedKeys, state.Key)
				}
				optionMap[state.Key] = options.WithOptionSet{Validator: legacy, Check: check}
			}
			issues := options.ValidateOptionsMap(context.Background(), &mockAPI{}, entry, entry.Fields[2], optionMap)
			if legacyCalls != len(observed) || !reflect.DeepEqual(invocationOrder, expectedKeys) {
				t.Errorf("legacy calls=%d observed=%d; check order=%v want %v", legacyCalls, len(observed), invocationOrder, expectedKeys)
			}
			var states, findings strings.Builder
			for _, state := range observed {
				fmt.Fprintf(&states, "%s has-value=%t value=%q\n", state.Key, state.HasVal, state.Value)
			}
			for _, issue := range issues {
				fmt.Fprintf(&findings, "%s: %s\n", issue.Level, issue.Message)
			}
			if states.String() != sections["expected/options.txt"] {
				t.Errorf("options:\n%s\nwant:\n%s", states.String(), sections["expected/options.txt"])
			}
			if findings.String() != sections["expected/findings.txt"] {
				t.Errorf("findings:\n%s\nwant:\n%s", findings.String(), sections["expected/findings.txt"])
			}
			if entry.RawLine != originalRaw || !reflect.DeepEqual(entry.Fields, originalFields) {
				t.Error("verification changed source")
			}
			var formatted bytes.Buffer
			if err := format.Format(strings.NewReader(input), &formatted); err != nil {
				t.Fatal(err)
			}
			roundTrip, err := core.ParseFstab(&formatted)
			if err != nil || len(roundTrip) != 1 || roundTrip[0].Fields[3] != originalFields[3] {
				t.Fatalf("format changed options: entries=%v error=%v", roundTrip, err)
			}
		})
	}
}

func TestOptionSetIsolationAndZeroValue(t *testing.T) {
	var empty options.OptionSet
	if empty.Has("missing") || len(empty.Ordered()) != 0 || len(empty.All("missing")) != 0 {
		t.Fatal("zero value is not empty")
	}
	if _, ok := empty.Last("missing"); ok {
		t.Fatal("missing last value reported present")
	}
	set := options.ParseOptionSet("flag,flag=,flag=a=b,other")
	if !set.Has("flag") || set.Has("missing") {
		t.Fatal("incorrect presence")
	}
	last, ok := set.Last("flag")
	if !ok || last.Value != "a=b" || !last.HasVal {
		t.Fatalf("last = %#v, %t", last, ok)
	}
	all, ordered := set.All("flag"), set.Ordered()
	all[0].Key, all[1].Value, ordered[2].Value = "changed", "changed", "changed"
	if got := set.All("flag"); len(got) != 3 || got[0].Key != "flag" || got[1].Value != "" || got[2].Value != "a=b" {
		t.Fatalf("accessor mutated shared observations: %#v", got)
	}
}

func TestWithOptionSetKeepsSyntaxAndDiagnosticOrder(t *testing.T) {
	entry := &rules.FstabEntry{LineNum: 4, RawLine: "test", Fields: []string{"test", "/test", "test", "second,first=,second=bad,first=ok,unknown"}}
	var order []string
	check := options.OptionSetValidatorFunc(func(_ context.Context, _ rules.API, _ *rules.FstabEntry, _, key string, _ options.OptionSet) []rules.ValidationIssue {
		order = append(order, key)
		return nil
	})
	optionMap := map[string]options.OptionValidator{
		"first":  options.WithOptionSet{Validator: options.StringOption{}, Check: check},
		"second": options.WithOptionSet{Validator: options.NoValueOption{}, Check: check},
	}
	issues := options.ValidateOptionsMap(context.Background(), &mockAPI{}, entry, "test", optionMap)
	if !reflect.DeepEqual(order, []string{"second", "first"}) {
		t.Fatalf("checks reordered: %v", order)
	}
	if len(issues) != 3 || !strings.Contains(issues[0].Message, "first") || !strings.Contains(issues[1].Message, "second") || !strings.Contains(issues[2].Message, "unknown") {
		t.Fatalf("per-occurrence syntax/unknown diagnostics changed: %v", issues)
	}
}

type optionsSliceFunc func([]options.OptionState) []rules.ValidationIssue

func (f optionsSliceFunc) ValidateOptions(_ context.Context, _ rules.API, _ *rules.FstabEntry, _ string, states []options.OptionState) []rules.ValidationIssue {
	return f(states)
}

func TestSliceValidatorsShareParserWithoutSharingMutableSlices(t *testing.T) {
	entry := &rules.FstabEntry{Fields: []string{"test", "/test", "test", `context="a,b",context=,context`}}
	want := options.ParseOptionSet(entry.Fields[3]).Ordered()
	validators := []options.OptionsValidator{
		optionsSliceFunc(func(states []options.OptionState) []rules.ValidationIssue { states[0].Value = "changed"; return nil }),
		optionsSliceFunc(func(states []options.OptionState) []rules.ValidationIssue {
			if !reflect.DeepEqual(states, want) {
				t.Errorf("slice checks lost source shapes or shared mutation: %#v", states)
			}
			return nil
		}),
	}
	options.ValidateOptionsSlice(context.Background(), &mockAPI{}, entry, "test", validators)
}
