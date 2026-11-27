package builtin_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin"
)

func TestFstabPermissionsRule(t *testing.T) {
	rule := &builtin.FstabPermissionsRule{}

	rootUID := 0
	rootGID := 0
	userUID := 1000
	userGID := 1000
	hasACL := true

	tests := []struct {
		name       string
		targetPath string
		info       rules.FileInfoResult
		wantIssues int
		wantMsg    string
	}{
		{
			name:       "root 0644 etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0644,
				UID:   &rootUID,
				GID:   &rootGID,
			},
			wantIssues: 0,
		},
		{
			name:       "root 0600 etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0600,
				UID:   &rootUID,
				GID:   &rootGID,
			},
			wantIssues: 0,
		},
		{
			name:       "world-writable etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0666,
				UID:   &rootUID,
				GID:   &rootGID,
			},
			wantIssues: 1,
			wantMsg:    "world-writable (mode 0666); recommended 0644 or 0600",
		},
		{
			name:       "non-root owner etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0644,
				UID:   &userUID,
				GID:   &userGID,
			},
			wantIssues: 1,
			wantMsg:    "owned by uid 1000, expected root (0); recommended root:root ownership",
		},
		{
			name:       "non-root group-writable etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0664,
				UID:   &rootUID,
				GID:   &userGID,
			},
			wantIssues: 1,
			wantMsg:    "group-writable by non-root gid 1000 (mode 0664)",
		},
		{
			name:       "root group-writable etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0664,
				UID:   &rootUID,
				GID:   &rootGID,
			},
			wantIssues: 0,
		},
		{
			name:       "alternate file with non-root owner",
			targetPath: "/tmp/my_fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0644,
				UID:   &userUID,
				GID:   &userGID,
			},
			wantIssues: 0,
		},
		{
			name:       "alternate path ending in /etc/fstab is not system fstab",
			targetPath: "/tmp/test/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0644,
				UID:   &userUID,
				GID:   &userGID,
			},
			wantIssues: 0,
		},
		{
			name:       "alternate file world-writable",
			targetPath: "/tmp/my_fstab",
			info: rules.FileInfoResult{
				State: rules.Exists,
				Mode:  0666,
				UID:   &userUID,
				GID:   &userGID,
			},
			wantIssues: 1,
			wantMsg:    "world-writable (mode 0666)",
		},
		{
			name:       "ACL write exposure on etc fstab",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State:       rules.Exists,
				Mode:        0644,
				UID:         &rootUID,
				GID:         &rootGID,
				HasWriteACL: &hasACL,
			},
			wantIssues: 1,
			wantMsg:    "ACL granting write access to non-root users",
		},
		{
			name:       "unknown metadata",
			targetPath: "/etc/fstab",
			info: rules.FileInfoResult{
				State: rules.Unknown,
			},
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &testFileInfoAPI{info: tt.info}
			ctx := rules.WithTargetFstab(context.Background(), tt.targetPath)
			entry1 := &rules.FstabEntry{LineNum: 1, RawLine: "LABEL=root / ext4 defaults 0 1"}
			entry2 := &rules.FstabEntry{LineNum: 2, RawLine: "LABEL=boot /boot ext4 defaults 0 2"}

			issues1, err := rule.Validate(ctx, api, entry1)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}
			if len(issues1) != tt.wantIssues {
				t.Fatalf("got %d issues on line 1, want %d", len(issues1), tt.wantIssues)
			}
			if tt.wantIssues > 0 && tt.wantMsg != "" {
				if !contains(issues1[0].Message, tt.wantMsg) {
					t.Errorf("got message %q, want containing %q", issues1[0].Message, tt.wantMsg)
				}
				if issues1[0].Level != rules.IssueLevelWarning {
					t.Errorf("got level %v, want warning", issues1[0].Level)
				}
			}

			// Validate that second entry does not duplicate issues
			issues2, err := rule.Validate(ctx, api, entry2)
			if err != nil {
				t.Fatalf("Validate on line 2 failed: %v", err)
			}
			if len(issues2) != 0 {
				t.Errorf("line 2 duplicated %d issues, want 0", len(issues2))
			}
		})
	}
}

func TestFstabPermissionsRule_ValidateFile_EmptyFstab(t *testing.T) {
	rule := &builtin.FstabPermissionsRule{}
	rootUID := 0
	rootGID := 0
	api := &testFileInfoAPI{
		info: rules.FileInfoResult{
			State: rules.Exists,
			Mode:  0666,
			UID:   &rootUID,
			GID:   &rootGID,
		},
	}
	ctx := rules.WithTargetFstab(context.Background(), "/etc/fstab")
	issues, err := rule.ValidateFile(ctx, api)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if !contains(issues[0].Message, "world-writable") {
		t.Fatalf("unexpected message: %q", issues[0].Message)
	}
}

type testFileInfoAPI struct {
	info rules.FileInfoResult
}

func (t *testFileInfoAPI) CheckFileExists(path string) rules.FileExistsResult {
	return rules.FileExists
}
func (t *testFileInfoAPI) CheckBlockDeviceExists(dev string) rules.ExistState {
	return rules.Exists
}
func (t *testFileInfoAPI) GetHostKernelVersion() string {
	return "6.6.0"
}
func (t *testFileInfoAPI) CheckUserExists(uid int) rules.ExistState {
	return rules.Exists
}
func (t *testFileInfoAPI) CheckGroupExists(gid int) rules.ExistState {
	return rules.Exists
}
func (t *testFileInfoAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{Available: true}, nil
}
func (t *testFileInfoAPI) ResolveDevicePath(dev string) string {
	return dev
}
func (t *testFileInfoAPI) GetFileInfo(path string) (rules.FileInfoResult, error) {
	return t.info, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (findSubstr(s, substr))))
}

func findSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
