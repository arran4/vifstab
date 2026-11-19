package builtin

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

// SystemdOptionsRule checks the generator namespace across all filesystem types.
// It is independent of optional post-save systemctl integration.
type SystemdOptionsRule struct{}

func init() { rules.Register(&SystemdOptionsRule{}) }

func (*SystemdOptionsRule) Name() string           { return "systemd-options" }
func (*SystemdOptionsRule) Dependencies() []string { return nil }
func (*SystemdOptionsRule) Validate(ctx context.Context, _ rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return options.ValidateSystemdOptions(ctx, entry), nil
}
