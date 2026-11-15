package editor

import (
	"errors"
	"os"
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
