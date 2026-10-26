package txtar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/rules"
	x_txtar "golang.org/x/tools/txtar"
)

func fixtureAPI() *rules.CacheFS {
	c := rules.NewCache()
	c.Set("proc_version", "Linux version 5.10.0 (PRIVATE_BUILDER@PRIVATE_MACHINE) custom-build")
	c.Set("proc_filesystems", map[string]bool{"tmpfs": true, "ext4": true, "btrfs": true, "vfat": true, "overlay": true, "cifs": true, "nfs": true, "zfs": true})
	c.Set("block_devices", rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{{Path: "/dev/PRIVATE_DISK", UUID: "PRIVATE_UUID", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", MountPoints: []string{"/boot/efi", "/mnt/PRIVATE_MOUNT"}, DeviceID: "PRIVATE_DEVICE_ID", SysfsName: "PRIVATE_SYSFS", Aliases: []string{"/dev/disk/by-uuid/PRIVATE_UUID"}}}, Diagnostics: []rules.BlockDeviceDiagnostic{{Source: "PRIVATE_DIAGNOSTIC_PATH", Error: "PRIVATE_DIAGNOSTIC_ERROR"}}})
	c.Set("block_device_exists:/dev/PRIVATE_DISK", rules.Exists)
	c.Set("block_device_exists:UUID=PRIVATE_UUID", rules.Exists)
	c.Set("block_device_exists:/dev/PRIVATE_ABSENT", rules.DoesNotExist)
	c.Set("block_device_exists:/dev/PRIVATE_DENIED", rules.PermissionDenied)
	c.Set("user_exists:1234", rules.DoesNotExist)
	c.Set("group_exists:2345", rules.PermissionDenied)
	c.Set("file_exists:/PRIVATE_MISSING", rules.FileDoesNotExist)
	c.Set("file_exists:/PRIVATE_DENIED", rules.FilePermissionDenied)
	c.Set("file_exists:/PRIVATE_UNKNOWN", rules.Unknown)
	a := rules.NewCacheFS(c, fstest.MapFS{})
	a.OfflineReplay = true
	return a
}
func captureText(t *testing.T, api rules.API, text string, raw bool) []byte {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(p, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := CaptureSystem(context.Background(), api, p, raw)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func validation(t *testing.T, api rules.API, b []byte) []rules.ValidationIssue {
	t.Helper()
	v, err := editor.VerifyAndLintReader(bytes.NewReader(b), api)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestProductionValidationRoundTrip(t *testing.T) {
	cases := map[string]struct {
		fstab, finding string
		blocking       bool
	}{
		"valid":          {"tmpfs /tmp tmpfs defaults 0 0\n", "", false},
		"malformed":      {"malformed\n", "", true},
		"partition-role": {"UUID=PRIVATE_UUID /boot/efi ext4 defaults 0 2\n", "GPT partition type is EFI System Partition (ESP) but filesystem is not FAT", true},
		"kernel-option":  {"/dev/PRIVATE_DISK /mnt/PRIVATE_TARGET btrfs compress=zstd 0 2\n", "", true},
		"accounts":       {"/dev/PRIVATE_DISK /mnt/PRIVATE_TARGET vfat uid=1234,gid=2345 0 2\n", "does not exist", true},
		"files":          {"overlay /mnt/PRIVATE_TARGET overlay lowerdir=/PRIVATE_MISSING,upperdir=/PRIVATE_DENIED,workdir=/PRIVATE_UNKNOWN 0 0\n", "does not exist", true},
		"devices":        {"/dev/PRIVATE_DISK /mnt/PRIVATE_TARGET btrfs device=/dev/PRIVATE_ABSENT 0 2\n", "does not exist", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			before := validation(t, fixtureAPI(), []byte(tc.fstab))
			for _, raw := range []bool{true, false} {
				archive := captureText(t, fixtureAPI(), tc.fstab, raw)
				s, err := ParseTXTAR(archive)
				if err != nil {
					t.Fatal(err)
				}
				api, err := ReplaySnapshot(s)
				if err != nil {
					t.Fatal(err)
				}
				after := validation(t, api, s.Fstab)
				if raw && !reflect.DeepEqual(before, after) {
					t.Fatalf("findings changed on unredacted round trip:\nbefore=%#v\nafter=%#v", before, after)
				}
				blocked := false
				for _, v := range after {
					blocked = blocked || rules.IsBlockingIssue(v)
				}
				// Compare the production blocking policy after redaction as well.
				if blocked != tc.blocking {
					t.Fatalf("blocking=%v, want %v: %#v", blocked, tc.blocking, after)
				}
				if tc.finding != "" && !strings.Contains(fmt.Sprint(after), tc.finding) {
					t.Fatalf("missing production finding %q: %#v", tc.finding, after)
				}
				if !raw && strings.Contains(string(archive), "PRIVATE_") {
					t.Fatalf("private value leaked: %s", archive)
				}
			}
		})
	}
}
func TestKernelDependentFindingRoundTrip(t *testing.T) {
	api := fixtureAPI()
	api.Cache.Set("proc_version", "Linux version 4.13.0")
	text := "/dev/PRIVATE_DISK /tmp btrfs compress=zstd 0 2\n"
	before := validation(t, api, []byte(text))
	if !strings.Contains(fmt.Sprint(before), "4.14") {
		t.Fatalf("expected kernel-dependent finding: %#v", before)
	}
	s, err := ParseTXTAR(captureText(t, api, text, true))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if after := validation(t, replay, s.Fstab); !reflect.DeepEqual(before, after) {
		t.Fatalf("kernel finding changed: %#v vs %#v", before, after)
	}
}
func TestInventoryRoundTripAndDeterminism(t *testing.T) {
	text := "UUID=PRIVATE_UUID /boot/efi ext4 defaults 0 2\n"
	for _, raw := range []bool{true, false} {
		a := captureText(t, fixtureAPI(), text, raw)
		b := captureText(t, fixtureAPI(), text, raw)
		if !bytes.Equal(a, b) {
			t.Fatal("capture serialization is nondeterministic")
		}
		s, err := ParseTXTAR(a)
		if err != nil {
			t.Fatal(err)
		}
		api, err := ReplaySnapshot(s)
		if err != nil {
			t.Fatal(err)
		}
		inv, err := api.GetBlockDevices()
		if err != nil {
			t.Fatal(err)
		}
		original, _ := fixtureAPI().GetBlockDevices()
		if raw && !reflect.DeepEqual(inv, original) {
			t.Fatalf("inventory changed: %#v != %#v", inv, original)
		}
		if len(inv.Diagnostics) != 1 || len(inv.Devices[0].MountPoints) != 2 {
			t.Fatal("redaction lost inventory information")
		}
		if !raw && strings.Contains(string(a), "PRIVATE_") {
			t.Fatalf("privacy leak: %s", a)
		}
	}
}
func TestIncompleteReplayNeverInfersAbsence(t *testing.T) {
	text := `-- input/etc/fstab --
overlay /mnt overlay lowerdir=/UNRECORDED,upperdir=/OTHER,workdir=/THIRD 0 0
/dev/unknown /boot/efi ext4 defaults 0 2
none /external zfs defaults 0 0
/dev/unknown /data btrfs subvol=/UNRECORDED 0 2
`
	s, err := ParseTXTAR([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	api, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if api.GetHostKernelVersion() != "" {
		t.Fatal("live kernel fallback")
	}
	for _, ref := range []string{"/dev/unknown", "/dev/ubda", "mtd:test", "/etc/passwd", "/proc/version"} {
		if api.CheckBlockDeviceExists(ref) != rules.Unknown || api.CheckFileExists(ref) != rules.Unknown {
			t.Fatalf("invented observation for %s", ref)
		}
	}
	if api.CheckUserExists(0) != rules.Unknown || api.CheckGroupExists(0) != rules.Unknown {
		t.Fatal("live account fallback")
	}
	if inv, err := api.GetBlockDevices(); err != nil || inv.Available {
		t.Fatalf("live inventory fallback: %#v %v", inv, err)
	}
	for _, v := range validation(t, api, s.Fstab) {
		if rules.IsBlockingIssue(v) || strings.Contains(v.Message, "does not exist") {
			t.Fatalf("unknown became absence: %#v", v)
		}
	}
}
func TestFileAndBlockWithoutOtherObservation(t *testing.T) {
	s := &Snapshot{Cache: map[string]string{"file_exists:/dev/test": "true"}}
	api, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if api.CheckBlockDeviceExists("/dev/test") != rules.Unknown {
		t.Fatal("file observation used as block observation")
	}
	s = &Snapshot{Cache: map[string]string{"block_exists:/dev/test": "true"}}
	api, err = ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if api.CheckFileExists("/dev/test") != rules.Unknown {
		t.Fatal("block observation used as file observation")
	}
}
func TestStrictArchiveValidation(t *testing.T) {
	base := "-- input/etc/fstab --\n"
	cases := map[string]string{
		"empty": "", "absolute": "-- /etc/fstab --\n", "traversal": "-- input/a/../b --\n", "double-slash": "-- input//a --\n", "backslash": "-- input\\a --\n",
		"alias-duplicate": base + "-- input --\n", "too-long": base + "-- " + strings.Repeat("a", MaxNameLength+1) + " --\n",
		"section-size":      base + strings.Repeat("a", MaxSectionSize+1),
		"invalid-inventory": base + "-- provider/inventory.json --\n[not-json]", "null-inventory": base + "-- provider/inventory.json --\nnull",
		"null-observations": base + "-- provider/observations.json --\nnull", "invalid-state": base + "-- provider/observations.json --\n{\"files\":{\"a\":77}}",
		"invalid-account":          base + "-- provider/observations.json --\n{\"users\":{\"alice\":0}}",
		"manifest-duplicate-field": base + "-- manifest.json --\n{\"version\":\"1.0\",\"version\":\"2.0\",\"type\":\"system\",\"redacted\":true}",
		"manifest-unknown-field":   base + "-- manifest.json --\n{\"version\":\"1.0\",\"type\":\"system\",\"redacted\":true,\"extra\":true}",
		"manifest-trailing":        base + "-- manifest.json --\n{\"version\":\"1.0\",\"type\":\"system\",\"redacted\":true} {}",
		"cache-conflict":           base + "-- cache --\nfile_exists:/test=true\n-- cache --\nfile_exists:/test=false\n",
		"cache-malformed":          base + "-- cache --\nnot-an-observation\n", "cache-invalid-state": base + "-- cache --\nblock_exists:/test=perhaps\n",
		"legacy-invalid-inventory": base + "-- cache --\nblock_devices=garbage\n",
	}
	sections := base
	for i := 0; i < MaxSectionCount; i++ {
		sections += fmt.Sprintf("-- section%d --\n", i)
	}
	cases["too-many-sections"] = sections
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseTXTAR([]byte(data)); err == nil {
				t.Fatal("unsafe/malformed archive accepted")
			}
		})
	}
	if _, err := Read(io.LimitReader(zeroReader{}, MaxArchiveSize+1)); err == nil {
		t.Fatal("unbounded input accepted")
	}
	// A harmless '..' within a component is not traversal.
	if _, err := ParseTXTAR([]byte(base + "-- input/a..b --\n")); err != nil {
		t.Fatal(err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestRedactedEscapedAndQuotedIdentities(t *testing.T) {
	inv := rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{
		{Path: "/dev/PRIVATE_ONE", UUID: "ABCDEF-1234", Label: "PRIVATE LABEL", Aliases: []string{`/dev/disk/by-label/PRIVATE\x20LABEL`, `/dev/disk/by-partlabel/PRIVATE\x20PART`, "/dev/disk/by-id/PRIVATE_ID"}},
		{Path: "/dev/PRIVATE_TWO", UUID: "PRIVATE_SECOND", Label: "PRIVATE_OTHER"},
	}}
	lines := []string{`UUID="abcdef-1234"`, `LABEL='PRIVATE\040LABEL'`, `/dev/disk/by-label/PRIVATE\x20LABEL`, `PARTLABEL=PRIVATE\040PART`, `ID="PRIVATE_ID"`}
	var text strings.Builder
	for _, ref := range lines {
		fmt.Fprintf(&text, "%s /mnt/PRIVATE_TARGET ext4 defaults 0 2\n", ref)
	}
	b := captureText(t, multiDeviceAPI{inventory: inv}, text.String(), false)
	if strings.Contains(string(b), "PRIVATE") || strings.Contains(string(b), "ABCDEF") {
		t.Fatalf("identifier leak: %s", b)
	}
	s, err := ParseTXTAR(b)
	if err != nil {
		t.Fatal(err)
	}
	api, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := api.GetBlockDevices()
	for _, line := range strings.Split(strings.TrimSpace(string(s.Fstab)), "\n") {
		ref := strings.Fields(line)[0]
		d := got.Resolve(ref)
		if d == nil || d.Path != got.Devices[0].Path {
			t.Errorf("reference %q did not resolve to first device: %#v", ref, d)
		}
	}
}
func TestCaptureDoesNotMutateInventory(t *testing.T) {
	api := fixtureAPI()
	before, _ := api.GetBlockDevices()
	b, _ := json.Marshal(before)
	captureText(t, api, "UUID=PRIVATE_UUID /tmp ext4 defaults 0 2\n", false)
	after, _ := api.GetBlockDevices()
	a, _ := json.Marshal(after)
	if !bytes.Equal(a, b) {
		t.Fatal("capture mutated caller's inventory")
	}
}
func TestCaptureBoundsAndDelimiterInjection(t *testing.T) {
	for name, text := range map[string]string{"oversize": strings.Repeat("a", MaxSectionSize+1), "line-limit": strings.Repeat("a", 100000), "delimiter": "tmpfs /tmp tmpfs defaults 0 0\n-- input/etc/fstab --\n"} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "fstab")
			if err := os.WriteFile(p, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := CaptureSystem(context.Background(), fixtureAPI(), p, true); err == nil {
				t.Fatal("invalid capture accepted")
			}
		})
	}
	// Generated structured sections must round-trip as the expected number of files.
	b := captureText(t, fixtureAPI(), "tmpfs /tmp tmpfs defaults 0 0\n", false)
	if len(x_txtar.Parse(b).Files) != 5 {
		t.Fatal("unexpected generated section count")
	}
}

func TestProviderUnavailableRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		inv rules.BlockDeviceInventory
		err error
	}{
		{inv: rules.BlockDeviceInventory{Available: false, Diagnostics: []rules.BlockDeviceDiagnostic{{Source: "PRIVATE_SOURCE", Error: "PRIVATE_ERROR"}}}},
		{err: fmt.Errorf("PRIVATE_PROVIDER_IO_ERROR")},
	} {
		for _, raw := range []bool{true, false} {
			b := captureText(t, multiDeviceAPI{inventory: tc.inv, err: tc.err}, "/dev/UNKNOWN /boot/efi ext4 defaults 0 2\n", raw)
			s, err := ParseTXTAR(b)
			if err != nil {
				t.Fatal(err)
			}
			api, err := ReplaySnapshot(s)
			if err != nil {
				t.Fatal(err)
			}
			inv, err := api.GetBlockDevices()
			if inv.Available || (err != nil) != (tc.err != nil) {
				t.Fatalf("availability changed: %#v %v", inv, err)
			}
			if tc.err == nil && len(inv.Diagnostics) != len(tc.inv.Diagnostics) {
				t.Fatal("diagnostic availability lost")
			}
			if !raw && strings.Contains(string(b), "PRIVATE_") {
				t.Fatalf("unavailability leaks details: %s", b)
			}
			if _, err := rules.SupportedFilesystems(api); err == nil {
				t.Fatal("missing support became known")
			}
		}
	}
}
func TestKnownEmptySupportIsNotUnknown(t *testing.T) {
	api := fixtureAPI()
	api.Cache.Set("proc_filesystems", map[string]bool{})
	b := captureText(t, api, "none /tmp ext4 defaults 0 0\n", true)
	s, err := ParseTXTAR(b)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	m, err := rules.SupportedFilesystems(replay)
	if err != nil || m == nil || len(m) != 0 {
		t.Fatalf("explicit empty support lost: %#v %v", m, err)
	}
	if !strings.Contains(fmt.Sprint(validation(t, replay, s.Fstab)), "Unknown filesystem type") {
		t.Fatal("explicit negative support was ignored")
	}
}
func TestNullAndConflictingObservationsRejected(t *testing.T) {
	for _, tail := range []string{
		"-- provider/observations.json --\n{\"files\":{\"/exists\":null}}\n",
		"-- provider/observations.json --\n{\"files\":{},\"Files\":{}}\n",
		"-- cache --\nblock_exists:/test=true\nblock_device_exists:/test=false\n",
		"-- cache --\nproc_filesystems=ext4\n-- provider/observations.json --\n{\"filesystems\":{\"ext4\":false}}\n",
		"-- input/provider/inventory.json --\n{}\n-- provider/inventory.json --\n{}\n",
	} {
		if _, err := ParseTXTAR([]byte("-- input/etc/fstab --\n" + tail)); err == nil {
			t.Fatalf("conflicting observation accepted: %s", tail)
		}
	}
	// Filesystem paths are case-sensitive and can coexist.
	if _, err := ParseTXTAR([]byte("-- input/etc/fstab --\n-- provider/observations.json --\n{\"files\":{\"/a\":0,\"/A\":1}}\n")); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyContextObservationUsesProductionPipeline(t *testing.T) {
	b := []byte("-- input/etc/fstab --\nnone /tmp btrfs subvol=/missing 0 2\n-- kernel --\n6.5.0\n-- cache --\nproc_filesystems=btrfs\nbtrfs_subvol_exists:/missing=false\n")
	s, err := ParseTXTAR(b)
	if err != nil {
		t.Fatal(err)
	}
	api, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fmt.Sprint(validation(t, api, s.Fstab)), "Subvolume '/missing' might not exist") {
		t.Fatal("explicit fixture observation not supplied to production rules")
	}
}

func TestExactBlockObservationPrecedesResolvedFallback(t *testing.T) {
	s := &Snapshot{Observations: &Observations{
		Blocks: map[string]rules.ExistState{"UUID=test": rules.Unknown, "/dev/disk/by-uuid/test": rules.Exists},
	}}
	api, err := ReplaySnapshot(s)
	if err != nil {
		t.Fatal(err)
	}
	if api.CheckBlockDeviceExists("UUID=test") != rules.Unknown {
		t.Fatal("resolved observation replaced explicit unknown")
	}
	if api.CheckBlockDeviceExists("/dev/disk/by-uuid/test") != rules.Exists {
		t.Fatal("exact alias observation lost")
	}
}
