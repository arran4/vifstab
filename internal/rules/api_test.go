package rules_test

import (
	"testing"
	"github.com/arran4/vifstab/internal/rules"
)

func TestGetBlockDevices(t *testing.T) {
	// A basic sanity check for ActualFS
	api := rules.ActualFS{}
	devs, err := api.GetBlockDevices()
	if err != nil {
		t.Errorf("GetBlockDevices failed: %v", err)
	}

	// Just check it ran without error and potentially returned some results
	// (depends on the host environment, but usually won't be empty)
	if len(devs) == 0 {
		t.Logf("GetBlockDevices returned no devices. This may be normal in an isolated test environment.")
	} else {
		t.Logf("Found %d block devices.", len(devs))
	}
}
