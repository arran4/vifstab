package editor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
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

type XattrRemover interface {
	Removexattr(path, name string) error
}

type TargetMetadata struct {
	Mode              os.FileMode
	UID               int
	GID               int
	HasUID            bool
	HasGID            bool
	Xattrs            map[string][]byte
	XattrsObserved    bool
	XattrsUnsupported bool
	XattrErr          error
	HasXattrProvider  bool
}

func extractUIDGID(info os.FileInfo) (int, int, bool) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid), true
	}
	if ms, ok := info.(interface{ MockSys() (int, int, bool) }); ok {
		return ms.MockSys()
	}
	return 0, 0, false
}

func isXattrUnsupported(err error) bool {
	return errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}

func extractTargetMetadata(path string, info os.FileInfo, fsLayer FS) TargetMetadata {
	meta := TargetMetadata{
		Mode: info.Mode(),
	}
	if uid, gid, has := extractUIDGID(info); has {
		meta.UID = uid
		meta.GID = gid
		meta.HasUID = true
		meta.HasGID = true
	}

	if xh, ok := fsLayer.(XattrHandler); ok {
		meta.HasXattrProvider = true
		attrs, err := xh.Listxattr(path)
		if err != nil {
			if isXattrUnsupported(err) {
				meta.XattrsUnsupported = true
			} else {
				meta.XattrErr = fmt.Errorf("failed to list extended attributes on %q: %w", path, err)
			}
		} else {
			meta.XattrsObserved = true
			if len(attrs) > 0 {
				meta.Xattrs = make(map[string][]byte, len(attrs))
				for _, attr := range attrs {
					val, err := xh.Getxattr(path, attr)
					if err != nil {
						meta.XattrErr = fmt.Errorf("failed to read extended attribute %q on %q: %w", attr, path, err)
						break
					}
					meta.Xattrs[attr] = val
				}
			}
		}
	}
	return meta
}

func policyFailure(policy config.MetadataPolicy, msg string) (string, error) {
	if policy == config.MetadataPolicyBestEffort {
		return msg, nil
	}
	if policy == config.MetadataPolicyStrict {
		return "", fmt.Errorf("%s (strict policy refuses replacement; edited content preserved; use --metadata-policy=supported or best-effort to override)", msg)
	}
	return "", errors.New(msg)
}

func applyAndVerifyMetadata(tmpPath string, tmpDest TempFile, meta TargetMetadata, policy config.MetadataPolicy, fsLayer FS) ([]string, error) {
	if policy == "" {
		policy = config.MetadataPolicyStrict
	}
	var warnings []string
	recordIssue := func(msg string) error {
		w, err := policyFailure(policy, msg)
		if err != nil {
			return err
		}
		if w != "" {
			warnings = append(warnings, w)
		}
		return nil
	}

	// 1. Ownership (UID/GID)
	if !meta.HasUID || !meta.HasGID {
		if policy == config.MetadataPolicyStrict {
			return nil, fmt.Errorf("cannot observe destination ownership (UID/GID) (strict policy refuses replacement; edited content preserved; use --metadata-policy=supported or best-effort to override)")
		}
		warnings = append(warnings, "destination ownership (UID/GID) unavailable from provider; ownership cannot be verified")
	} else {
		var chownErr error
		chownSupported := false
		if tfc, ok := tmpDest.(TempFileChowner); ok {
			chownSupported = true
			chownErr = tfc.Chown(meta.UID, meta.GID)
		} else if ch, ok := fsLayer.(Chowner); ok {
			chownSupported = true
			chownErr = ch.Chown(tmpPath, meta.UID, meta.GID)
		}
		if !chownSupported {
			if err := recordIssue("ownership preservation provider unavailable"); err != nil {
				return nil, err
			}
		} else if chownErr != nil {
			if err := recordIssue(fmt.Sprintf("failed to preserve destination ownership (%d:%d): %v", meta.UID, meta.GID, chownErr)); err != nil {
				return nil, err
			}
		}
	}

	// 2. Mode (including special bits) - applied after chown so setuid/setgid are not cleared
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
		if err := recordIssue(fmt.Sprintf("failed to chmod temporary destination file: %v", chmodErr)); err != nil {
			return nil, err
		}
	}

	// 3. Extended attributes / ACLs / Security labels
	if !meta.HasXattrProvider {
		if policy == config.MetadataPolicyStrict {
			return nil, fmt.Errorf("extended attribute/ACL/security-label provider unavailable (strict policy refuses replacement; edited content preserved; use --metadata-policy=supported or best-effort to override)")
		}
		warnings = append(warnings, "extended attribute/ACL/security-label provider unavailable; xattrs/ACLs/security labels not preserved")
	} else if meta.XattrErr != nil {
		if err := recordIssue(meta.XattrErr.Error()); err != nil {
			return nil, err
		}
	} else if meta.XattrsUnsupported {
		if policy == config.MetadataPolicyStrict {
			return nil, fmt.Errorf("filesystem does not support extended attributes/ACLs/security labels (strict policy refuses replacement; edited content preserved; use --metadata-policy=supported or best-effort to override)")
		}
		warnings = append(warnings, fmt.Sprintf("filesystem does not support extended attributes/ACLs/security labels; continuing under %s policy", policy))
	} else if meta.XattrsObserved {
		xh := fsLayer.(XattrHandler)
		if len(meta.Xattrs) > 0 {
			keys := make([]string, 0, len(meta.Xattrs))
			for k := range meta.Xattrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, attr := range keys {
				if err := xh.Setxattr(tmpPath, attr, meta.Xattrs[attr]); err != nil {
					if rErr := recordIssue(fmt.Sprintf("failed to preserve extended attribute %q: %v", attr, err)); rErr != nil {
						return nil, rErr
					}
				}
			}
		}

		// Clean up any inherited destination xattrs not present on source
		if tmpAttrs, err := xh.Listxattr(tmpPath); err == nil {
			for _, attr := range tmpAttrs {
				if _, expected := meta.Xattrs[attr]; !expected {
					if xr, ok := fsLayer.(XattrRemover); ok {
						if rmErr := xr.Removexattr(tmpPath, attr); rmErr != nil {
							if rErr := recordIssue(fmt.Sprintf("failed to remove inherited extended attribute %q from temporary destination: %v", attr, rmErr)); rErr != nil {
								return nil, rErr
							}
						}
					} else {
						if rErr := recordIssue(fmt.Sprintf("temporary destination has inherited extended attribute %q not present on source and removal is unsupported", attr)); rErr != nil {
							return nil, rErr
						}
					}
				}
			}
		} else if !isXattrUnsupported(err) {
			if rErr := recordIssue(fmt.Sprintf("failed to list temporary destination extended attributes for cleanup: %v", err)); rErr != nil {
				return nil, rErr
			}
		}
	}

	// 4. Independent Reverification
	tmpInfo, err := fsLayer.Stat(tmpPath)
	if err != nil {
		if rErr := recordIssue(fmt.Sprintf("failed to re-stat temporary destination file for metadata verification: %v", err)); rErr != nil {
			return nil, rErr
		}
		return warnings, nil
	}

	if tmpInfo.Mode().Perm() != meta.Mode.Perm() {
		if rErr := recordIssue(fmt.Sprintf("destination mode reverification failed: got %04o, want %04o", tmpInfo.Mode().Perm(), meta.Mode.Perm())); rErr != nil {
			return nil, rErr
		}
	}

	origSpecial := meta.Mode & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	gotSpecial := tmpInfo.Mode() & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	if origSpecial != gotSpecial {
		if rErr := recordIssue(fmt.Sprintf("destination special mode bits dropped: got %v, want %v", gotSpecial, origSpecial)); rErr != nil {
			return nil, rErr
		}
	}

	if meta.HasUID && meta.HasGID {
		gotUID, gotGID, hasGot := extractUIDGID(tmpInfo)
		if !hasGot {
			if rErr := recordIssue("destination ownership reverification unavailable on temporary file"); rErr != nil {
				return nil, rErr
			}
		} else if gotUID != meta.UID || gotGID != meta.GID {
			if rErr := recordIssue(fmt.Sprintf("destination ownership reverification failed: got %d:%d, want %d:%d", gotUID, gotGID, meta.UID, meta.GID)); rErr != nil {
				return nil, rErr
			}
		}
	}

	if meta.HasXattrProvider && meta.XattrsObserved && meta.XattrErr == nil && !meta.XattrsUnsupported {
		xh := fsLayer.(XattrHandler)
		gotAttrs, listErr := xh.Listxattr(tmpPath)
		if listErr != nil {
			if rErr := recordIssue(fmt.Sprintf("extended attribute reverification list failed: %v", listErr)); rErr != nil {
				return nil, rErr
			}
		} else {
			gotSet := make(map[string]struct{}, len(gotAttrs))
			for _, a := range gotAttrs {
				gotSet[a] = struct{}{}
				if _, expected := meta.Xattrs[a]; !expected {
					if rErr := recordIssue(fmt.Sprintf("extended attribute reverification found unexpected attribute %q", a)); rErr != nil {
						return nil, rErr
					}
				}
			}
			keys := make([]string, 0, len(meta.Xattrs))
			for k := range meta.Xattrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if _, ok := gotSet[k]; !ok {
					if rErr := recordIssue(fmt.Sprintf("extended attribute reverification missing %q", k)); rErr != nil {
						return nil, rErr
					}
					continue
				}
				gotVal, getErr := xh.Getxattr(tmpPath, k)
				if getErr != nil {
					if rErr := recordIssue(fmt.Sprintf("extended attribute reverification read failed for %q: %v", k, getErr)); rErr != nil {
						return nil, rErr
					}
					continue
				}
				if !bytes.Equal(gotVal, meta.Xattrs[k]) {
					if rErr := recordIssue(fmt.Sprintf("extended attribute reverification value mismatch for %q", k)); rErr != nil {
						return nil, rErr
					}
				}
			}
		}
	}

	return warnings, nil
}
