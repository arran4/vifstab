package editor

import (
	"github.com/arran4/vifstab/internal/rules"
	"testing"
)

type countingAPI struct {
	rules.API
	count int
}

func (c *countingAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	c.count++
	return rules.BlockDeviceInventory{}, nil
}

func TestMemoizedAPI_GetBlockDevices(t *testing.T) {
	api := &countingAPI{}
	memoized := &memoizedAPI{API: api}

	_, _ = memoized.GetBlockDevices()
	_, _ = memoized.GetBlockDevices()

	if api.count != 1 {
		t.Errorf("GetBlockDevices called %d times, expected 1", api.count)
	}
}

func TestMemoizedAPI_NoCall(t *testing.T) {
	api := &countingAPI{}
	memoized := &memoizedAPI{API: api}
	_ = memoized

	if api.count != 0 {
		t.Errorf("GetBlockDevices called %d times, expected 0", api.count)
	}
}
