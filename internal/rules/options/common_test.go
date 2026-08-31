package options_test

import (
	"testing"

	"github.com/arran4/vifstab/internal/rules/options"
)

func TestCommonOptions(t *testing.T) {
	opts := options.CommonOptions()

	expectedKeys := []string{"defaults", "rw", "ro", "noatime", "relatime", "suid", "nosuid", "exec", "noexec", "user", "nouser", "context"}
	for _, key := range expectedKeys {
		if _, ok := opts[key]; !ok {
			t.Errorf("expected CommonOptions() to contain %q", key)
		}
	}
}
