//go:build linux

package rules

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func buildPOSIXACLPayload(version uint32, entries [][3]uint32) []byte {
	buf := make([]byte, 4+8*len(entries))
	binary.LittleEndian.PutUint32(buf[:4], version)
	for i, e := range entries {
		off := 4 + i*8
		binary.LittleEndian.PutUint16(buf[off:off+2], uint16(e[0]))
		binary.LittleEndian.PutUint16(buf[off+2:off+4], uint16(e[1]))
		binary.LittleEndian.PutUint32(buf[off+4:off+8], e[2])
	}
	return buf
}

func TestParsePOSIXAccessACLWrite(t *testing.T) {
	owner := 1000
	tests := []struct {
		name      string
		data      []byte
		ownerUID  *int
		wantWrite bool
		wantOK    bool
	}{
		{
			name: "read-only named user",
			data: buildPOSIXACLPayload(posixACLXattrVersion, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
				{uint32(aclUser), 4, 1001},
				{uint32(aclGroupObj), 4, 0xFFFFFFFF},
				{uint32(aclMask), 6, 0xFFFFFFFF},
				{uint32(aclOther), 4, 0xFFFFFFFF},
			}),
			ownerUID:  &owner,
			wantWrite: false,
			wantOK:    true,
		},
		{
			name: "writable named user",
			data: buildPOSIXACLPayload(posixACLXattrVersion, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
				{uint32(aclUser), 6, 1001},
				{uint32(aclGroupObj), 4, 0xFFFFFFFF},
				{uint32(aclMask), 6, 0xFFFFFFFF},
				{uint32(aclOther), 4, 0xFFFFFFFF},
			}),
			ownerUID:  &owner,
			wantWrite: true,
			wantOK:    true,
		},
		{
			name: "named user write masked out by ACL_MASK",
			data: buildPOSIXACLPayload(posixACLXattrVersion, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
				{uint32(aclUser), 6, 1001},
				{uint32(aclGroupObj), 4, 0xFFFFFFFF},
				{uint32(aclMask), 4, 0xFFFFFFFF},
				{uint32(aclOther), 4, 0xFFFFFFFF},
			}),
			ownerUID:  &owner,
			wantWrite: false,
			wantOK:    true,
		},
		{
			name: "named group write for non-root gid",
			data: buildPOSIXACLPayload(posixACLXattrVersion, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
				{uint32(aclGroupObj), 4, 0xFFFFFFFF},
				{uint32(aclGroup), 2, 100},
				{uint32(aclMask), 6, 0xFFFFFFFF},
				{uint32(aclOther), 4, 0xFFFFFFFF},
			}),
			ownerUID:  &owner,
			wantWrite: true,
			wantOK:    true,
		},
		{
			name: "named user matching owner UID is ignored",
			data: buildPOSIXACLPayload(posixACLXattrVersion, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
				{uint32(aclUser), 6, 1000},
				{uint32(aclGroupObj), 4, 0xFFFFFFFF},
				{uint32(aclMask), 6, 0xFFFFFFFF},
				{uint32(aclOther), 4, 0xFFFFFFFF},
			}),
			ownerUID:  &owner,
			wantWrite: false,
			wantOK:    true,
		},
		{
			name:     "truncated header",
			data:     []byte{2, 0},
			ownerUID: &owner,
			wantOK:   false,
		},
		{
			name: "wrong version",
			data: buildPOSIXACLPayload(99, [][3]uint32{
				{uint32(aclUserObj), 6, 0xFFFFFFFF},
			}),
			ownerUID: &owner,
			wantOK:   false,
		},
		{
			name:     "truncated entry",
			data:     []byte{2, 0, 0, 0, 1, 0, 6},
			ownerUID: &owner,
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotWrite, gotOK := parsePOSIXAccessACLWrite(tc.data, tc.ownerUID)
			if gotOK != tc.wantOK || gotWrite != tc.wantWrite {
				t.Fatalf("parsePOSIXAccessACLWrite() = (%v, %v), want (%v, %v)", gotWrite, gotOK, tc.wantWrite, tc.wantOK)
			}
		})
	}
}

func TestActualFS_GetFileInfo_LinuxRealFS(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fstab.target")
	if err := os.WriteFile(target, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0666); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "fstab.link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	fs := ActualFS{}
	info, err := fs.GetFileInfo(link)
	if err != nil {
		t.Fatalf("GetFileInfo(symlink) error: %v", err)
	}
	if info.State != Exists {
		t.Fatalf("State = %v, want Exists", info.State)
	}
	if info.Mode.Perm() != 0666 {
		t.Fatalf("Mode.Perm() = %04o, want 0666 (target mode via symlink)", info.Mode.Perm())
	}
	if info.UID == nil || info.GID == nil {
		t.Fatalf("expected non-nil UID/GID on Linux real FS")
	}
	// On filesystems without ACLs (ENODATA), HasWriteACL is false; if xattrs are
	// unsupported (ENOTSUP), HasWriteACL is nil (unknown). It must never be true here.
	if info.HasWriteACL != nil && *info.HasWriteACL {
		t.Fatalf("expected HasWriteACL to be false or nil on plain 0666 file, got true")
	}
}
