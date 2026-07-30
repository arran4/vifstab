package options_test

import (
	"github.com/arran4/vifstab/internal/rules"
)

type mockAPI struct {
	rules.API
	existingUsers  map[int]bool
	existingGroups map[int]bool
	kernelVerStr   string
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

func (m *mockAPI) GetHostKernelVersion() string {
	return m.kernelVerStr
}
