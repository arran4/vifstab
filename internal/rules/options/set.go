package options

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// OptionSet preserves the order and every occurrence produced by SplitOptions.
// Its zero value is empty. Accessors return copies so a check cannot change the
// observations seen by another check. It does not interpret overriding options.
type OptionSet struct {
	ordered []OptionState
	byKey   map[string][]OptionState
	first   map[string]int
}

// ParseOptionSet shares the existing quote/escape handling of rules.SplitOptions.
// In particular, a flag and an explicitly empty value remain different shapes.
func ParseOptionSet(raw string) OptionSet {
	set := OptionSet{byKey: make(map[string][]OptionState), first: make(map[string]int)}
	for _, rawOption := range rules.SplitOptions(raw) {
		key, value, hasValue := strings.Cut(rawOption, "=")
		option := OptionState{Key: key, Value: value, HasVal: hasValue}
		if !set.Has(key) {
			set.first[key] = len(set.ordered)
		}
		set.ordered = append(set.ordered, option)
		set.byKey[key] = append(set.byKey[key], option)
	}
	return set
}

// Ordered returns all occurrences in source order.
func (s OptionSet) Ordered() []OptionState { return append([]OptionState(nil), s.ordered...) }

// All returns every occurrence of key in source order.
func (s OptionSet) All(key string) []OptionState {
	return append([]OptionState(nil), s.byKey[key]...)
}

// Has reports whether key occurs, including bare flags and empty values.
func (s OptionSet) Has(key string) bool { return len(s.byKey[key]) != 0 }

// FirstIndex returns a key's first position in Ordered, for once-per-key checks.
func (s OptionSet) FirstIndex(key string) (int, bool) {
	i, ok := s.first[key]
	return i, ok
}

// Last returns the final occurrence of key, for explicitly last-value policies.
func (s OptionSet) Last(key string) (OptionState, bool) {
	values := s.byKey[key]
	if len(values) == 0 {
		return OptionState{}, false
	}
	return values[len(values)-1], true
}

// OptionSetValidator checks a registered key once, at its first occurrence. It
// can inspect the entire entry, including earlier and later repeated values.
// Per-occurrence syntax checks still use OptionValidator.
type OptionSetValidator interface {
	ValidateOptionSet(context.Context, rules.API, *rules.FstabEntry, string, string, OptionSet) []rules.ValidationIssue
}

type OptionSetValidatorFunc func(context.Context, rules.API, *rules.FstabEntry, string, string, OptionSet) []rules.ValidationIssue

func (f OptionSetValidatorFunc) ValidateOptionSet(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName, key string, set OptionSet) []rules.ValidationIssue {
	return f(ctx, api, entry, fsName, key, set)
}

// WithOptionSet adds a whole-entry check to an existing per-occurrence validator.
// Register this as the outermost validator in the option map. ValidateOptionsMap
// runs Check once per key; directly calling ValidateOption only runs Validator.
type WithOptionSet struct {
	Validator OptionValidator
	Check     OptionSetValidator
}

func (w WithOptionSet) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName, key, value string, hasValue bool, legacy map[string]OptionState) []rules.ValidationIssue {
	if w.Validator == nil {
		return nil
	}
	return w.Validator.ValidateOption(ctx, api, entry, fsName, key, value, hasValue, legacy)
}

func (w WithOptionSet) ValidateOptionSet(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName, key string, set OptionSet) []rules.ValidationIssue {
	if w.Check == nil {
		return nil
	}
	return w.Check.ValidateOptionSet(ctx, api, entry, fsName, key, set)
}
