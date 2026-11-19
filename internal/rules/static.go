package rules

import "errors"

// ErrStaticObservation identifies deliberately unchecked live observations.
var ErrStaticObservation = errors.New("static verification: live observation unchecked")

// StaticAPI permits the host-independent portion of rules to run while every
// live observation remains unknown. It contains no OS or fallback provider.
type StaticAPI struct{}

var _ API = StaticAPI{}
var _ FilesystemSupportProvider = StaticAPI{}

func (StaticAPI) CheckFileExists(string) FileExistsResult  { return Unknown }
func (StaticAPI) CheckBlockDeviceExists(string) ExistState { return Unknown }
func (StaticAPI) CheckUserExists(int) ExistState           { return Unknown }
func (StaticAPI) CheckGroupExists(int) ExistState          { return Unknown }
func (StaticAPI) GetHostKernelVersion() string             { return "" }
func (StaticAPI) ResolveDevicePath(ref string) string      { return ResolveDevicePath(ref) }
func (StaticAPI) GetBlockDevices() (BlockDeviceInventory, error) {
	return BlockDeviceInventory{Available: false}, ErrStaticObservation
}
func (StaticAPI) GetSupportedFilesystems() (map[string]bool, error) { return nil, ErrStaticObservation }
