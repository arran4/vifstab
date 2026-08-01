package vfat

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
)

type VfatRule struct{}

func init() {
	rules.Register(&VfatRule{})
}

func (r *VfatRule) Name() string {
	return "vfat"
}

func (r *VfatRule) Dependencies() []string {
	return []string{}
}

func (r *VfatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vfat" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := rules.SplitOptions(entry.Fields[3])

	// Keep track of the original fields string for fatRule call
	originalOpts := entry.Fields[3]
	newOptsList := []string{}

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if key == "dotsOK" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("vfat explicitly kills dotsOK mount option on line %d", entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
			continue
		}

		// vfat specific options
		vfatSpecific := false
		switch key {
		case "uni_xlate", "posix", "nonumtail", "utf8":
			vfatSpecific = true
			if val != "" {
				if val != "0" && val != "1" && val != "yes" && val != "no" && val != "true" && val != "false" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for vfat mount option '%s' on line %d, must be a boolean (0, 1, yes, no, true, false)", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "shortname":
			vfatSpecific = true
			if val != "" {
				if val != "lower" && val != "win95" && val != "winnt" && val != "mixed" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'shortname' on line %d", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			} else {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for vfat mount option 'shortname' on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		if !vfatSpecific {
			newOptsList = append(newOptsList, opt)
		}
	}

	// Delegate to fat options
	fatRule := &fat.FatRule{}
	mockEntry := *entry
	mockEntry.Fields = append([]string{}, entry.Fields...)
	mockEntry.Fields[2] = "fat"
	mockEntry.Fields[3] = strings.Join(newOptsList, ",")

	fatIssues, err := fatRule.Validate(ctx, api, &mockEntry)
	if err != nil {
		return issues, err
	}

	// Adjust messages to say 'vfat' instead of 'fat'
	for _, issue := range fatIssues {
		issue.Message = strings.ReplaceAll(issue.Message, "fat mount option", "vfat mount option")
		issues = append(issues, issue)
	}

	entry.Fields[3] = originalOpts
	return issues, nil
}
