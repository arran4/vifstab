package rules

import (
	"context"
	"testing"
)

type dummyRule struct {
	name string
}

func (d *dummyRule) Name() string {
	return d.name
}

func (d *dummyRule) Dependencies() []string {
	return nil
}

func (d *dummyRule) Validate(ctx context.Context, api API, entry *FstabEntry) ([]ValidationIssue, error) {
	return nil, nil
}

func TestRegisterAndGetAllRules(t *testing.T) {
	initialRules := GetAllRules()
	initialCount := len(initialRules)

	testRule := &dummyRule{name: "test_dummy_rule_123"}
	Register(testRule)

	newRules := GetAllRules()
	if len(newRules) != initialCount+1 {
		t.Errorf("Expected %d rules after registration, got %d", initialCount+1, len(newRules))
	}

	found := false
	for _, r := range newRules {
		if r.Name() == "test_dummy_rule_123" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Newly registered rule was not found in the output of GetAllRules()")
	}
}
