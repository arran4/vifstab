package builtin

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/arran4/vifstab/internal/rules"
)

type FstabPermissionsRule struct{}

func init() {
	rules.Register(&FstabPermissionsRule{})
}

func (r *FstabPermissionsRule) Name() string {
	return "fstab_permissions"
}

func (r *FstabPermissionsRule) Dependencies() []string {
	return nil
}

func (r *FstabPermissionsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	cache := rules.CacheFromContext(ctx)
	if cache != nil {
		if _, ok := cache.Get("fstab_permissions_checked"); ok {
			return nil, nil
		}
		cache.Set("fstab_permissions_checked", true)
	} else if entry.LineNum > 1 {
		return nil, nil
	}
	return r.validateTarget(ctx, api, entry.RawLine)
}

func (r *FstabPermissionsRule) ValidateFile(ctx context.Context, api rules.API) ([]rules.ValidationIssue, error) {
	cache := rules.CacheFromContext(ctx)
	if cache != nil {
		if _, ok := cache.Get("fstab_permissions_checked"); ok {
			return nil, nil
		}
		cache.Set("fstab_permissions_checked", true)
	}
	return r.validateTarget(ctx, api, "")
}

func (r *FstabPermissionsRule) validateTarget(ctx context.Context, api rules.API, rawLine string) ([]rules.ValidationIssue, error) {
	targetPath := rules.TargetFstabFromContext(ctx)
	if targetPath == "" {
		targetPath = "/etc/fstab"
	}

	info, err := api.GetFileInfo(targetPath)
	if err != nil || info.State != rules.Exists {
		// Unknown or unavailable metadata means unknown, not insecure or safe.
		return nil, nil
	}

	var issues []rules.ValidationIssue

	isSystemTarget := isSystemFstab(targetPath)

	// Check unexpected owner for system target
	if isSystemTarget && info.UID != nil && *info.UID != 0 {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("fstab target '%s' is owned by uid %d, expected root (0); recommended root:root ownership", targetPath, *info.UID),
			Line:    rawLine,
			Level:   rules.IssueLevelWarning,
		})
	}

	// Check world-writable permissions
	perm := info.Mode.Perm()
	if perm&0002 != 0 {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("fstab target '%s' is world-writable (mode %04o); recommended 0644 or 0600", targetPath, perm),
			Line:    rawLine,
			Level:   rules.IssueLevelWarning,
		})
	}

	// Check group-writable by non-root group on system target
	if isSystemTarget && (perm&0020 != 0) && info.GID != nil && *info.GID != 0 {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("fstab target '%s' is group-writable by non-root gid %d (mode %04o); recommended removing group write permission", targetPath, *info.GID, perm),
			Line:    rawLine,
			Level:   rules.IssueLevelWarning,
		})
	}

	// Check ACL write exposure
	if info.HasWriteACL != nil && *info.HasWriteACL {
		msg := fmt.Sprintf("fstab target '%s' has ACL granting write access to non-root users; recommended removing write ACL", targetPath)
		if !isSystemTarget {
			msg = fmt.Sprintf("fstab target '%s' has ACL granting write access to non-owner users; recommended removing write ACL", targetPath)
		}
		issues = append(issues, rules.ValidationIssue{
			Message: msg,
			Line:    rawLine,
			Level:   rules.IssueLevelWarning,
		})
	}

	return issues, nil
}

func isSystemFstab(path string) bool {
	clean := filepath.Clean(path)
	return clean == "/etc/fstab" || clean == "etc/fstab"
}
