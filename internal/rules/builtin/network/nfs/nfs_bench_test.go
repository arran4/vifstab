package nfs

import (
	"context"
	"github.com/arran4/vifstab/internal/rules"
	"testing"
)

type mockAPI struct{}

func (m mockAPI) Register(rule rules.Rule)                           {}
func (m mockAPI) Rules() []rules.Rule                                { return nil }
func (m mockAPI) CheckBlockDeviceExists(path string) bool            { return true }
func (m mockAPI) CheckFileExists(path string) rules.FileExistsResult { return rules.FileExists }
func (m mockAPI) CheckUserExists(username int) bool                  { return true }
func (m mockAPI) CheckGroupExists(groupname int) bool                { return true }
func (mockAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{}, nil
}
func (mockAPI) ResolveDevicePath(dev string) string             { return dev }
func (m mockAPI) GetHostKernelVersion() string                  { return "5.10.0" }
func (m mockAPI) IsVirtualMachine() bool                        { return false }
func (m mockAPI) IsContainer() bool                             { return false }
func (m mockAPI) GetKernelModuleInfo(name string) (bool, error) { return true, nil }

func BenchmarkNfsOptions(b *testing.B) {
	rule := &NfsRule{}
	entry1 := &rules.FstabEntry{Fields: []string{"server:/path", "/mnt", "nfs", "rw"}}
	entry2 := &rules.FstabEntry{Fields: []string{"server:/path", "/mnt", "nfs4", "rw"}}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule.Validate(ctx, mockAPI{}, entry1)
		rule.Validate(ctx, mockAPI{}, entry2)
	}
}
