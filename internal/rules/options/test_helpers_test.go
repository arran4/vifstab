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

func (m *mockAPI) CheckFileExists(path string) rules.FileExistsResult {
	if m.existingFiles == nil {
		return rules.FileDoesNotExist
	}
	if m.existingFiles[path] {
		return rules.FileExists
	}
	return rules.FileDoesNotExist
}

func (m *mockAPI) CheckUserExists(uid int) rules.ExistState {
	if m.existingUsers == nil {
		return rules.DoesNotExist
	}
	if m.existingUsers[uid] {
		return rules.Exists
	} else {
		return rules.DoesNotExist
	}
}

func (m *mockAPI) CheckGroupExists(gid int) rules.ExistState {
	if m.existingGroups == nil {
		return rules.DoesNotExist
	}
	if m.existingGroups[gid] {
		return rules.Exists
	} else {
		return rules.DoesNotExist
	}
}

func (m *mockAPI) CheckBlockDeviceExists(dev string) rules.ExistState {
	if m.existingDevs == nil {
		return rules.DoesNotExist
	}
	if m.existingDevs[dev] {
		return rules.Exists
	}
	resolved := rules.ResolveDevicePath(dev)
	if m.existingDevs[resolved] {
		return rules.Exists
	} else {
		return rules.DoesNotExist
	}
}

func (m *mockAPI) GetHostKernelVersion() string {
	return m.kernelVerStr
}

func (m *mockAPI) ResolveDevicePath(dev string) string {
	return rules.ResolveDevicePath(dev)
}

func (m *mockAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{}, nil
}
