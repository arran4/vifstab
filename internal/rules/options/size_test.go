package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestSizeOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 18}
	opt := options.SizeOption{AllowPercentage: true}

	validSizes := []string{"1024", "64k", "512M", "4G", "50%"}
	for _, size := range validSizes {
		t.Run("valid size "+size, func(t *testing.T) {
			issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", size, true, nil)
			if len(issues) != 0 {
				t.Errorf("expected 0 issues for %q, got %d", size, len(issues))
			}
		})
	}

	t.Run("invalid percentage size", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "150%", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue for 150%%, got %d", len(issues))
		}
	})

	t.Run("invalid size string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "invalidsize", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue for 'invalidsize', got %d", len(issues))
		}
	})
}
