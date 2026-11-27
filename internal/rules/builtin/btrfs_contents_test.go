package builtin_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
)

func TestBtrfsSubvolumeContentsRemainUnknownWithoutObservation(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing")
	if err := os.Mkdir(existing, 0700); err != nil {
		t.Fatal(err)
	}
	var rule rules.Rule
	for _, candidate := range rules.GetAllRules() {
		if normalizeRuleName(candidate.Name()) == "btrfs" {
			rule = candidate
		}
	}
	if rule == nil {
		t.Fatal("Btrfs rule missing")
	}
	for _, path := range []string{existing, filepath.Join(root, "missing")} {
		cache := rules.NewCache()
		cache.Set("proc_version", "Linux version 6.5.0")
		ctx := rules.WithCache(context.Background(), cache)
		api := rules.NewCacheFS(cache, fstest.MapFS{})
		entry := &rules.FstabEntry{LineNum: 1, RawLine: "fixture", Fields: []string{"UUID=fixture", "/target", "btrfs", "subvol=" + path}}
		issues, err := rule.Validate(ctx, api, entry)
		if err != nil || len(issues) != 0 {
			t.Fatalf("host VFS must not decide block filesystem contents: %v %v", issues, err)
		}
		cache.Set(rules.CacheKey("btrfs_subvol_exists:"+path), false)
		issues, err = rule.Validate(ctx, api, entry)
		if err != nil || len(issues) != 1 {
			t.Fatalf("explicit offline observation lost: %v %v", issues, err)
		}
	}
}
