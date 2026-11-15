package editor

import (
	"fmt"
	"os"
	"syscall"

	"github.com/arran4/vifstab/internal/config"
)

type Chowner interface {
	Chown(name string, uid, gid int) error
}

type TempFileChowner interface {
	Chown(uid, gid int) error
}

type XattrHandler interface {
	Getxattr(path, name string) ([]byte, error)
	Setxattr(path, name string, value []byte) error
	Listxattr(path string) ([]string, error)
}

type TargetMetadata struct {
	Mode   os.FileMode
	UID    int
	GID    int
	HasUID bool
	HasGID bool
	Xattrs map[string][]byte
}

func extractTargetMetadata(path string, info os.FileInfo, fsLayer FS) TargetMetadata {
	meta := TargetMetadata{
		Mode: info.Mode(),
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		meta.UID = int(stat.Uid)
		meta.GID = int(stat.Gid)
		meta.HasUID = true
		meta.HasGID = true
	} else if ms, ok := info.(interface{ MockSys() (int, int, bool) }); ok {
		uid, gid, has := ms.MockSys()
		if has {
			meta.UID = uid
			meta.GID = gid
			meta.HasUID = true
			meta.HasGID = true
		}
	}

	if xh, ok := fsLayer.(XattrHandler); ok {
		if attrs, err := xh.Listxattr(path); err == nil && len(attrs) > 0 {
			meta.Xattrs = make(map[string][]byte, len(attrs))
			for _, attr := range attrs {
				if val, err := xh.Getxattr(path, attr); err == nil {
					meta.Xattrs[attr] = val
				}
			}
		}
	}
	return meta
}

func applyAndVerifyMetadata(tmpPath string, tmpDest TempFile, meta TargetMetadata, policy config.MetadataPolicy, fsLayer FS) ([]string, error) {
	if policy == "" {
		policy = config.MetadataPolicyStrict
	}
	var warnings []string

	// 1. Ownership (UID/GID)
	if meta.HasUID && meta.HasGID {
		var chownErr error
		if tfc, ok := tmpDest.(TempFileChowner); ok {
			chownErr = tfc.Chown(meta.UID, meta.GID)
		} else if ch, ok := fsLayer.(Chowner); ok {
			chownErr = ch.Chown(tmpPath, meta.UID, meta.GID)
		}
		if chownErr != nil {
			switch policy {
			case config.MetadataPolicyStrict:
				return nil, fmt.Errorf("failed to preserve destination ownership (%d:%d): %w (strict policy refuses replacement; edited content preserved; use --metadata-policy=supported or best-effort to override)", meta.UID, meta.GID, chownErr)
			case config.MetadataPolicySupported:
				return nil, fmt.Errorf("failed to preserve destination ownership (%d:%d): %w", meta.UID, meta.GID, chownErr)
			case config.MetadataPolicyBestEffort:
				warnings = append(warnings, fmt.Sprintf("failed to preserve destination ownership (%d:%d): %v", meta.UID, meta.GID, chownErr))
			}
		}
	}

	// 2. Mode (including special bits)
	targetMode := meta.Mode.Perm()
	if meta.Mode&os.ModeSetuid != 0 {
		targetMode |= os.ModeSetuid
	}
	if meta.Mode&os.ModeSetgid != 0 {
		targetMode |= os.ModeSetgid
	}
	if meta.Mode&os.ModeSticky != 0 {
		targetMode |= os.ModeSticky
	}
	if chmodErr := tmpDest.Chmod(targetMode); chmodErr != nil {
		switch policy {
		case config.MetadataPolicyStrict, config.MetadataPolicySupported:
			return nil, fmt.Errorf("failed to chmod temporary destination file: %w", chmodErr)
		case config.MetadataPolicyBestEffort:
			warnings = append(warnings, fmt.Sprintf("failed to chmod temporary destination file: %v", chmodErr))
		}
	}

	// 3. Extended attributes / ACLs
	if len(meta.Xattrs) > 0 {
		if xh, ok := fsLayer.(XattrHandler); ok {
			for attr, val := range meta.Xattrs {
				if err := xh.Setxattr(tmpPath, attr, val); err != nil {
					switch policy {
					case config.MetadataPolicyStrict:
						return nil, fmt.Errorf("failed to preserve extended attribute %q: %w (strict policy refuses replacement)", attr, err)
					case config.MetadataPolicySupported:
						return nil, fmt.Errorf("failed to preserve extended attribute %q: %w", attr, err)
					case config.MetadataPolicyBestEffort:
						warnings = append(warnings, fmt.Sprintf("failed to preserve extended attribute %q: %v", attr, err))
					}
				}
			}
		}
	}

	// 4. Reverification
	// Stat temporary file to ensure applied mode matches target
	tmpInfo, err := fsLayer.Stat(tmpPath)
	if err == nil {
		if tmpInfo.Mode().Perm() != meta.Mode.Perm() {
			errMismatch := fmt.Errorf("destination mode reverification failed: got %04o, want %04o", tmpInfo.Mode().Perm(), meta.Mode.Perm())
			if policy == config.MetadataPolicyBestEffort {
				warnings = append(warnings, errMismatch.Error())
			} else {
				return nil, errMismatch
			}
		}
		// Check special bits preservation
		origSpecial := meta.Mode & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		gotSpecial := tmpInfo.Mode() & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		if origSpecial != gotSpecial {
			errMismatch := fmt.Errorf("destination special mode bits dropped: got %v, want %v", gotSpecial, origSpecial)
			if policy == config.MetadataPolicyBestEffort {
				warnings = append(warnings, errMismatch.Error())
			} else {
				return nil, errMismatch
			}
		}
	}

	return warnings, nil
}
