package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// RequiresOption wraps an OptionValidator and requires that one or more other option keys are also present in mount options.
type RequiresOption struct {
	Validator    OptionValidator
	RequiredKeys []string
	CustomErrMsg string
}

func (r RequiresOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	var issues []rules.ValidationIssue

	if r.Validator != nil {
		issues = append(issues, r.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)...)
	}

	for _, reqKey := range r.RequiredKeys {
		if _, ok := allOpts[reqKey]; !ok {
			msg := r.CustomErrMsg
			if msg == "" {
				msg = fmt.Sprintf("%s mount option '%s' requires '%s' on line %d", fsName, key, reqKey, entry.LineNum)
			} else if strings.Contains(msg, "%d") {
				msg = fmt.Sprintf(msg, entry.LineNum)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: msg,
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	}

	return issues
}

// ConflictsOption wraps an OptionValidator and ensures that one or more conflicting option keys are NOT present in mount options.
type ConflictsOption struct {
	Validator       OptionValidator
	ConflictingKeys []string
	CustomErrMsg    string
}

func (c ConflictsOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	var issues []rules.ValidationIssue

	if c.Validator != nil {
		issues = append(issues, c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)...)
	}

	for _, confKey := range c.ConflictingKeys {
		if _, ok := allOpts[confKey]; ok {
			msg := c.CustomErrMsg
			if msg == "" {
				msg = fmt.Sprintf("%s mount option '%s' conflicts with '%s' on line %d", fsName, key, confKey, entry.LineNum)
			} else if strings.Contains(msg, "%d") {
				msg = fmt.Sprintf(msg, entry.LineNum)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: msg,
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	}

	return issues
}

// ConditionalOption evaluates a custom condition predicate against all parsed mount options.
type ConditionalOption struct {
	Validator OptionValidator
	Condition func(allOpts map[string]OptionState) bool
	ErrMsg    string
	Level     rules.IssueLevel
}

func (c ConditionalOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	var issues []rules.ValidationIssue

	if c.Validator != nil {
		issues = append(issues, c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)...)
	}

	if c.Condition != nil && !c.Condition(allOpts) {
		level := c.Level
		if level == "" {
			level = rules.IssueLevelError
		}
		msg := c.ErrMsg
		if strings.Contains(msg, "%d") {
			msg = fmt.Sprintf(msg, entry.LineNum)
		}
		issues = append(issues, rules.ValidationIssue{
			Message: msg,
			Line:    entry.RawLine,
			Level:   level,
		})
	}

	return issues
}
