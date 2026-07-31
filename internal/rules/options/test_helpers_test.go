package options_test

import (
	"github.com/arran4/vifstab/internal/rules"
)

type mockAPI struct {
	rules.API
	existingUsers  map[int]bool
	existingGroups map[int]bool
	existingDevs   map[string]bool
	existingFiles  map[string]bool
	kernelVerStr   string
}

func (m *mockAPI) CheckFileExists(path string) bool {
	if m.existingFiles == nil {
		return false
	}
	return m.existingFiles[path]
}

func (m *mockAPI) CheckUserExists(uid int) bool {
	if m.existingUsers == nil {
		return false
	}
	return m.existingUsers[uid]
}

func (m *mockAPI) CheckGroupExists(gid int) bool {
	if m.existingGroups == nil {
		return false
	}
	return m.existingGroups[gid]
}

func (m *mockAPI) CheckBlockDeviceExists(dev string) bool {
	if m.existingDevs == nil {
		return false
	}
	if m.existingDevs[dev] {
		return true
	}
	resolved := rules.ResolveDevicePath(dev)
	return m.existingDevs[resolved]
}

func (m *mockAPI) GetHostKernelVersion() string {
	return m.kernelVerStr
}
