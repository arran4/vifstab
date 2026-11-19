package nfs

import (
	"context"
	"github.com/arran4/vifstab/internal/rules"
	"testing"
)

type mockAPI struct{}

func (m mockAPI) Register(rule rules.Rule)                            {}
func (m mockAPI) Rules() []rules.Rule                                 { return nil }
func (m mockAPI) CheckBlockDeviceExists(path string) rules.ExistState { return rules.Exists }
func (m mockAPI) CheckFileExists(path string) rules.FileExistsResult  { return rules.FileExists }
func (m mockAPI) CheckUserExists(username int) rules.ExistState       { return rules.Exists }
func (m mockAPI) CheckGroupExists(groupname int) rules.ExistState     { return rules.Exists }
func (m mockAPI) GetHostKernelVersion() string                        { return "5.10.0" }
func (m mockAPI) IsVirtualMachine() bool                              { return false }
func (m mockAPI) IsContainer() bool                                   { return false }
func (m mockAPI) GetKernelModuleInfo(name string) (bool, error)       { return true, nil }
func (m mockAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{}, nil
}
func (m mockAPI) ResolveDevicePath(dev string) string { return rules.ResolveDevicePath(dev) }
func (m mockAPI) GetFileInfo(path string) (rules.FileInfoResult, error) {
	return rules.FileInfoResult{State: rules.Exists}, nil
}

func BenchmarkNfsOptions(b *testing.B) {
	rule := &NfsRule{}
	entry1 := &rules.FstabEntry{Fields: []string{"server:/path", "/mnt", "nfs", "rw"}}
	entry2 := &rules.FstabEntry{Fields: []string{"server:/path", "/mnt", "nfs4", "rw"}}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rule.Validate(ctx, mockAPI{}, entry1)
		_, _ = rule.Validate(ctx, mockAPI{}, entry2)
	}
}
