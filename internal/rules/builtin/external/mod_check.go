package external

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
)

func CheckModuleSupport(ctx context.Context, fsType string, entry *rules.FstabEntry, moduleName string, modNameAlternative string) []rules.ValidationIssue {
	// Module availability is optional, and none of these probes can establish
	// absence (a module can be unloaded or metadata inaccessible). There is no
	// validation finding to derive here. Avoid unrelated host reads in both live
	// verification and offline replay.
	return nil
}
