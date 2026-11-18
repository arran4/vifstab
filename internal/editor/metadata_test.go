package editor

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
)

type mockMetaFileInfo struct {
	os.FileInfo
	mode os.FileMode
	uid  int
	gid  int
}

func (m mockMetaFileInfo) Mode() os.FileMode { return m.mode }
func (m mockMetaFileInfo) MockSys() (int, int, bool) {
	return m.uid, m.gid, true
}

func TestMetadata_Strict_ChownFailure(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		faults: map[string]error{
			"destination.temp.chown": errors.New("operation not permitted"),
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyStrict,
	}

	newContent := []byte("LABEL=root / ext4 defaults,noatime 0 1\n")
	res := SaveAndVersion(cfg, newContent, fs)

	if res.Saved {
		t.Fatalf("expected save to be refused on chown failure under strict policy, but it succeeded")
	}
	if res.Err == nil {
		t.Fatalf("expected error on chown failure under strict policy, got nil")
	}

	// Verify original file is intact
	orig, err := fs.ReadFile(dest)
	if err != nil || string(orig) != "LABEL=root / ext4 defaults 0 1\n" {
		t.Errorf("original file was modified: %s (err: %v)", string(orig), err)
	}

	// Verify error message gives actionable diagnostic
	errMsg := res.Err.Error()
	if !containsStr(errMsg, "strict policy refuses replacement") {
		t.Errorf("expected diagnostic to mention strict policy refusal, got: %s", errMsg)
	}
	if !containsStr(errMsg, "edited content recovered at") {
		t.Errorf("expected recovery path in error message, got: %s", errMsg)
	}
}

func TestMetadata_Supported_ChownFailure(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		faults: map[string]error{
			"destination.temp.chown": errors.New("operation not permitted"),
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicySupported,
	}

	newContent := []byte("LABEL=root / ext4 defaults,noatime 0 1\n")
	res := SaveAndVersion(cfg, newContent, fs)

	if res.Saved {
		t.Fatalf("expected save to be refused on chown failure under supported policy, but it succeeded")
	}
	if res.Err == nil {
		t.Fatalf("expected error on chown failure, got nil")
	}

	// Original intact
	orig, err := fs.ReadFile(dest)
	if err != nil || string(orig) != "LABEL=root / ext4 defaults 0 1\n" {
		t.Errorf("original file was modified: %s (err: %v)", string(orig), err)
	}
}

func TestMetadata_BestEffort_ChownFailure(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		faults: map[string]error{
			"destination.temp.chown": errors.New("operation not permitted"),
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyBestEffort,
	}

	newContent := []byte("LABEL=root / ext4 defaults,noatime 0 1\n")
	res := SaveAndVersion(cfg, newContent, fs)

	if !res.Saved {
		t.Fatalf("expected save to succeed under best-effort policy despite chown failure, err: %v", res.Err)
	}
	if len(res.Warnings) == 0 {
		t.Errorf("expected warning about failed chown under best-effort, got 0 warnings")
	}

	// Destination should be updated
	updated, err := fs.ReadFile(dest)
	if err != nil || string(updated) != string(newContent) {
		t.Errorf("expected destination to be updated, got: %s (err: %v)", string(updated), err)
	}
}

func TestMetadata_Strict_ChmodFailure(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		faults: map[string]error{
			"destination.temp.chmod": errors.New("chmod failed"),
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyStrict,
	}

	res := SaveAndVersion(cfg, []byte("new content"), fs)
	if res.Saved {
		t.Fatalf("expected save to be refused on chmod failure, but it succeeded")
	}
	if res.Err == nil {
		t.Fatalf("expected error on chmod failure")
	}

	orig, _ := fs.ReadFile(dest)
	if string(orig) != "LABEL=root / ext4 defaults 0 1\n" {
		t.Errorf("original file was modified")
	}
}

func TestMetadata_Strict_XattrsPreserved(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		xattrs: map[string]map[string][]byte{
			"etc/fstab": {
				"security.selinux": []byte("system_u:object_r:etc_t:s0"),
			},
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyStrict,
	}

	newContent := []byte("LABEL=root / ext4 defaults,noatime 0 1\n")
	res := SaveAndVersion(cfg, newContent, fs)
	if !res.Saved {
		t.Fatalf("expected save to succeed with xattrs, got err: %v", res.Err)
	}

	// Verify xattrs were preserved on destination
	val, err := fs.Getxattr(dest, "security.selinux")
	if err != nil || string(val) != "system_u:object_r:etc_t:s0" {
		t.Errorf("xattr not preserved: got %q, err %v", string(val), err)
	}
}

func TestMetadata_Strict_XattrFailure(t *testing.T) {
	dest := "/etc/fstab"
	fs := &mockFS{
		mfs: fstest.MapFS{
			"etc/fstab": &fstest.MapFile{
				Data: []byte("LABEL=root / ext4 defaults 0 1\n"),
				Mode: 0644,
			},
		},
		xattrs: map[string]map[string][]byte{
			"etc/fstab": {
				"security.selinux": []byte("system_u:object_r:etc_t:s0"),
			},
		},
		faults: map[string]error{
			"destination.setxattr": errors.New("setxattr error"),
		},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyStrict,
	}

	res := SaveAndVersion(cfg, []byte("new content"), fs)
	if res.Saved {
		t.Fatalf("expected save to be refused on xattr failure under strict policy")
	}

	// But under best-effort:
	cfg.MetadataPolicy = config.MetadataPolicyBestEffort
	resBestEffort := SaveAndVersion(cfg, []byte("new content"), fs)
	if !resBestEffort.Saved {
		t.Fatalf("expected save to succeed under best-effort policy, got err: %v", resBestEffort.Err)
	}
	if len(resBestEffort.Warnings) == 0 {
		t.Errorf("expected warning for failed xattr under best-effort")
	}
}

func TestMetadata_NewFile_Default(t *testing.T) {
	dest := "/etc/fstab.new"
	fs := &mockFS{
		mfs: fstest.MapFS{},
	}

	cfg := config.Config{
		FilePath:       dest,
		MetadataPolicy: config.MetadataPolicyStrict,
	}

	res := SaveAndVersion(cfg, []byte("new file content"), fs)
	if !res.Saved {
		t.Fatalf("expected save to succeed for new file, got err: %v", res.Err)
	}
}

func TestMetadata_ListxattrAndGetxattrFailures(t *testing.T) {
	dest := "/etc/fstab"
	t.Run("listxattr EIO fails strict and supported", func(t *testing.T) {
		for _, pol := range []config.MetadataPolicy{config.MetadataPolicyStrict, config.MetadataPolicySupported} {
			fs := &mockFS{
				mfs: fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0644}},
				faults: map[string]error{
					"destination.listxattr": errors.New("I/O error"),
				},
			}
			res := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: pol}, []byte("new"), fs)
			if res.Saved || res.Err == nil || res.RecoveryPath == "" {
				t.Fatalf("policy %s: expected refused save with recovery, got %+v", pol, res)
			}
			if orig, _ := fs.ReadFile(dest); string(orig) != "orig" {
				t.Fatalf("original modified: %q", orig)
			}
		}
	})

	t.Run("listxattr ENOTSUP fails strict but warns under supported", func(t *testing.T) {
		fs := &mockFS{
			mfs: fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0644}},
			faults: map[string]error{
				"destination.listxattr": syscall.ENOTSUP,
			},
		}
		resStrict := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: config.MetadataPolicyStrict}, []byte("new"), fs)
		if resStrict.Saved || resStrict.Err == nil {
			t.Fatalf("expected strict to refuse on ENOTSUP, got %+v", resStrict)
		}

		resSup := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: config.MetadataPolicySupported}, []byte("new"), fs)
		if !resSup.Saved || len(resSup.Warnings) == 0 {
			t.Fatalf("expected supported to succeed with warning on ENOTSUP, got %+v", resSup)
		}
	})

	t.Run("getxattr failure fails strict and supported, warns best-effort", func(t *testing.T) {
		for _, pol := range []config.MetadataPolicy{config.MetadataPolicyStrict, config.MetadataPolicySupported} {
			fs := &mockFS{
				mfs:    fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0644}},
				xattrs: map[string]map[string][]byte{"etc/fstab": {"user.test": []byte("v")}},
				faults: map[string]error{"destination.getxattr": syscall.ENODATA},
			}
			res := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: pol}, []byte("new"), fs)
			if res.Saved || res.Err == nil {
				t.Fatalf("policy %s: expected failure on getxattr error, got %+v", pol, res)
			}
		}
		fsBest := &mockFS{
			mfs:    fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0644}},
			xattrs: map[string]map[string][]byte{"etc/fstab": {"user.test": []byte("v")}},
			faults: map[string]error{"destination.getxattr": syscall.ENODATA},
		}
		resBest := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: config.MetadataPolicyBestEffort}, []byte("new"), fsBest)
		if !resBest.Saved || len(resBest.Warnings) == 0 {
			t.Fatalf("expected best-effort to succeed with warning, got %+v", resBest)
		}
	})
}

type noUIDFS struct{ *mockFS }

func (f *noUIDFS) Stat(name string) (os.FileInfo, error) {
	info, err := f.mockFS.Stat(name)
	if err != nil {
		return nil, err
	}
	d := info.(dummyFileInfo)
	d.hasUID = false
	return d, nil
}
func (f *noUIDFS) Lstat(name string) (os.FileInfo, error) { return f.Stat(name) }

func TestMetadata_MissingUIDGID_And_SpecialBits_And_EmptyXattr(t *testing.T) {
	dest := "/etc/fstab"
	t.Run("missing UID/GID refuses strict and warns supported", func(t *testing.T) {
		fs := &noUIDFS{mockFS: &mockFS{mfs: fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0644}}}}
		resStrict := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: config.MetadataPolicyStrict}, []byte("new"), fs)
		if resStrict.Saved || resStrict.Err == nil {
			t.Fatalf("expected strict refusal when UID/GID unobservable, got %+v", resStrict)
		}
		resSup := SaveAndVersion(config.Config{FilePath: dest, MetadataPolicy: config.MetadataPolicySupported}, []byte("new"), fs)
		if !resSup.Saved || len(resSup.Warnings) == 0 {
			t.Fatalf("expected supported warning when UID/GID unobservable, got %+v", resSup)
		}
	})

	t.Run("special bits, ACLs, security labels, empty xattrs, and inherited cleanup", func(t *testing.T) {
		mode := os.FileMode(0640) | os.ModeSetgid | os.ModeSticky
		fs := &mockFS{
			mfs: fstest.MapFS{
				"etc/fstab":            {Data: []byte("orig"), Mode: mode},
				"etc/vifstab-tmp-mock": {Data: []byte{}, Mode: 0600},
			},
			uids: map[string]int{"etc/fstab": 0},
			gids: map[string]int{"etc/fstab": 15},
			xattrs: map[string]map[string][]byte{
				"etc/fstab": {
					"system.posix_acl_access": {0x02, 0x00, 0x00, 0x00},
					"security.selinux":        []byte("system_u:object_r:etc_t:s0\x00"),
					"user.empty":              {},
				},
				"etc/vifstab-tmp-mock": {
					"user.inherited_stale": []byte("stale"),
				},
			},
		}
		res := SaveAndVersion(config.Config{FilePath: dest, BackupDir: "/backup", MetadataPolicy: config.MetadataPolicyStrict}, []byte("new"), fs)
		if !res.Saved || !res.Versioned || res.Err != nil {
			t.Fatalf("expected full save and RCS version, got %+v", res)
		}
		info, err := fs.Stat(dest)
		if err != nil || info.Mode() != mode {
			t.Fatalf("mode mismatch: got %v want %v (err=%v)", info.Mode(), mode, err)
		}
		if _, exists := fs.xattrs["etc/fstab"]["user.inherited_stale"]; exists {
			t.Fatalf("inherited stale xattr was not removed")
		}
		if val, ok := fs.xattrs["etc/fstab"]["user.empty"]; !ok || len(val) != 0 {
			t.Fatalf("zero-length xattr not preserved: ok=%v val=%v", ok, val)
		}
	})
}

type dropSpecialBitsTempFile struct {
	TempFile
}

func (f *dropSpecialBitsTempFile) Chmod(mode os.FileMode) error {
	return f.TempFile.Chmod(mode & os.ModePerm) // silently drop setuid/setgid/sticky
}
func (f *dropSpecialBitsTempFile) Chown(uid, gid int) error {
	if c, ok := f.TempFile.(TempFileChowner); ok {
		return c.Chown(uid, gid)
	}
	return nil
}

type dropSpecialBitsFS struct{ *mockFS }

func (f *dropSpecialBitsFS) CreateTemp(dir, pattern string) (TempFile, error) {
	tf, err := f.mockFS.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &dropSpecialBitsTempFile{TempFile: tf}, nil
}

func TestMetadata_ReverificationDetectsDroppedSpecialBits(t *testing.T) {
	dest := "/etc/fstab"
	fs := &dropSpecialBitsFS{mockFS: &mockFS{
		mfs: fstest.MapFS{"etc/fstab": {Data: []byte("orig"), Mode: 0640 | os.ModeSetgid}},
	}}
	res := SaveAndVersion(config.Config{FilePath: dest, BackupDir: "/backup", MetadataPolicy: config.MetadataPolicyStrict}, []byte("new"), fs)
	if res.Saved || res.Versioned || res.Err == nil {
		t.Fatalf("expected reverification to catch dropped special bits, got %+v", res)
	}
	if !containsStr(res.Err.Error(), "special mode bits dropped") {
		t.Fatalf("unexpected error: %v", res.Err)
	}
}

func TestMetadata_RealFS_Linux(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fstab.real")
	link := filepath.Join(dir, "fstab")
	backupDir := filepath.Join(dir, "backup")

	if err := os.WriteFile(target, []byte("UUID=11111111-2222-4333-8444-555555555555 / ext4 defaults 0 1\n"), 0640); err != nil {
		t.Fatal(err)
	}
	wantMode := os.FileMode(0640) | os.ModeSticky
	if err := os.Chmod(target, wantMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	// Probe whether this filesystem supports user.* xattrs
	xattrSupported := true
	if err := osSetxattr(target, "user.vifstab_test", []byte("ok")); err != nil {
		if isXattrUnsupported(err) {
			xattrSupported = false
		}
	}

	policy := config.MetadataPolicyStrict
	if !xattrSupported {
		policy = config.MetadataPolicySupported
	}

	cfg := config.Config{
		FilePath:       link,
		BackupDir:      backupDir,
		MetadataPolicy: policy,
	}
	res := SaveAndVersion(cfg, []byte("UUID=11111111-2222-4333-8444-555555555555 / ext4 defaults,noatime 0 1\n"), DefaultFS)
	if !res.Saved || !res.Versioned || res.Err != nil {
		t.Fatalf("real FS save failed: %+v", res)
	}

	// Symlink must still be a symlink
	linfo, err := os.Lstat(link)
	if err != nil || linfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced by regular file: %v %v", linfo, err)
	}

	// Target mode & sticky bit preserved
	tinfo, err := os.Stat(target)
	if err != nil || tinfo.Mode() != wantMode {
		t.Fatalf("target mode = %v, want %v (err=%v)", tinfo.Mode(), wantMode, err)
	}

	if xattrSupported {
		val, err := osGetxattr(target, "user.vifstab_test")
		if err != nil || string(val) != "ok" {
			t.Fatalf("xattr not preserved on real FS: val=%q err=%v", val, err)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
