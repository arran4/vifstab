package autofs

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

func init() {
	rules.Register(&AutofsRule{})
}

type AutofsRule struct{}

func (r *AutofsRule) Name() string {
	return "autofs"
}

func (r *AutofsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *AutofsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) < 3 || entry.Fields[2] != "autofs" {
		return nil, nil
	}

	return []rules.ValidationIssue{
		{
			Message: fmt.Sprintf("autofs is managed by the automount daemon and should not be used as a client filesystem directly in fstab; use systemd.automount(5) or auto.master(5) instead on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		},
	}, nil
}
