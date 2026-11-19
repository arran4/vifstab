//go:build linux

package rules

import (
	"encoding/binary"
	"errors"

	"golang.org/x/sys/unix"
)

const (
	posixACLXattrName    = "system.posix_acl_access"
	posixACLXattrVersion = 2
	posixACLHeaderSize   = 4
	posixACLEntrySize    = 8

	aclUserObj  uint16 = 0x0001
	aclUser     uint16 = 0x0002
	aclGroupObj uint16 = 0x0004
	aclGroup    uint16 = 0x0008
	aclMask     uint16 = 0x0010
	aclOther    uint16 = 0x0020

	aclWrite uint16 = 0x02
)

// extractLinuxACL inspects the POSIX access ACL xattr on path when available.
// Missing ACLs (ENODATA) produce false; unsupported filesystems, permission
// errors, or malformed xattr payloads leave HasWriteACL nil (unknown).
func extractLinuxACL(path string, res *FileInfoResult) {
	if res == nil {
		return
	}
	buf, err := getLinuxXattr(path, posixACLXattrName)
	if err != nil {
		if errors.Is(err, unix.ENODATA) {
			f := false
			res.HasWriteACL = &f
		}
		return
	}
	hasWrite, ok := parsePOSIXAccessACLWrite(buf, res.UID)
	if !ok {
		return
	}
	res.HasWriteACL = &hasWrite
}

func getLinuxXattr(path, name string) ([]byte, error) {
	sz, err := unix.Getxattr(path, name, nil)
	if err != nil {
		return nil, err
	}
	if sz <= 0 {
		return nil, nil
	}
	buf := make([]byte, sz)
	n, err := unix.Getxattr(path, name, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// parsePOSIXAccessACLWrite parses a Linux system.posix_acl_access xattr payload
// and reports whether it grants effective write access via extended ACL entries
// (named user, named group, or other). Returns (false, false) on malformed data.
func parsePOSIXAccessACLWrite(data []byte, ownerUID *int) (hasWrite bool, ok bool) {
	if len(data) < posixACLHeaderSize || (len(data)-posixACLHeaderSize)%posixACLEntrySize != 0 {
		return false, false
	}
	version := binary.LittleEndian.Uint32(data[:posixACLHeaderSize])
	if version != posixACLXattrVersion {
		return false, false
	}

	type aclEntry struct {
		tag  uint16
		perm uint16
		id   uint32
	}

	count := (len(data) - posixACLHeaderSize) / posixACLEntrySize
	entries := make([]aclEntry, 0, count)
	mask := uint16(0xFFFF)
	hasMask := false

	for i := 0; i < count; i++ {
		off := posixACLHeaderSize + i*posixACLEntrySize
		e := aclEntry{
			tag:  binary.LittleEndian.Uint16(data[off : off+2]),
			perm: binary.LittleEndian.Uint16(data[off+2 : off+4]),
			id:   binary.LittleEndian.Uint32(data[off+4 : off+8]),
		}
		if e.tag == aclMask {
			mask = e.perm
			hasMask = true
		}
		entries = append(entries, e)
	}

	for _, e := range entries {
		switch e.tag {
		case aclUser:
			eff := e.perm
			if hasMask {
				eff &= mask
			}
			if eff&aclWrite != 0 {
				if ownerUID != nil && int(e.id) == *ownerUID {
					continue
				}
				return true, true
			}
		case aclGroup:
			eff := e.perm
			if hasMask {
				eff &= mask
			}
			if eff&aclWrite != 0 && e.id != 0 {
				return true, true
			}
		case aclOther:
			if e.perm&aclWrite != 0 {
				return true, true
			}
		}
	}

	return false, true
}
