package textinput

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type modeInfo struct{ mode fs.FileMode }

func (i modeInfo) Name() string       { return "fixture" }
func (i modeInfo) Size() int64        { return 0 }
func (i modeInfo) Mode() fs.FileMode  { return i.mode }
func (i modeInfo) ModTime() time.Time { return time.Time{} }
func (i modeInfo) IsDir() bool        { return i.mode.IsDir() }
func (i modeInfo) Sys() any           { return nil }

type fakeMetadata struct {
	info      fs.FileInfo
	statErr   error
	openErr   error
	closeErr  error
	reader    fs.File
	dataOpens int
	closes    int
}

func (m *fakeMetadata) Stat() (fs.FileInfo, error) { return m.info, m.statErr }
func (m *fakeMetadata) ReadOnly() (fs.File, error) {
	m.dataOpens++
	return m.reader, m.openErr
}
func (m *fakeMetadata) Close() error { m.closes++; return m.closeErr }

type fakeFile struct {
	info     fs.FileInfo
	statErr  error
	closeErr error
	closes   int
}

func (f *fakeFile) Read([]byte) (int, error)   { return 0, io.EOF }
func (f *fakeFile) Stat() (fs.FileInfo, error) { return f.info, f.statErr }
func (f *fakeFile) Close() error               { f.closes++; return f.closeErr }
func from(m metadata) func(string) (metadata, error) {
	return func(string) (metadata, error) { return m, nil }
}

func TestInputClassificationBeforeAnyDataOpen(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode fs.FileMode
		ok   bool
	}{
		{"regular", 0600, true},
		{"pipe", fs.ModeNamedPipe, true},
		{"character", fs.ModeDevice | fs.ModeCharDevice, true},
		{"block", fs.ModeDevice, false},
		{"directory", fs.ModeDir, false},
		{"socket", fs.ModeSocket, false},
		{"unclassified", fs.ModeIrregular, false},
		{"unresolved-link", fs.ModeSymlink, false},
		{"contradictory-stream", fs.ModeSocket | fs.ModeCharDevice, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &fakeMetadata{info: modeInfo{tc.mode}}
			if err := Check(m); (err == nil) != tc.ok {
				t.Fatalf("classification: %v", err)
			}
			if tc.ok {
				return // The native fixture tests exercise allowed descriptor opens.
			}
			_, err := open("/fixture", from(m))
			if err == nil || m.dataOpens != 0 || m.closes != 1 {
				t.Fatalf("refused input reached data open: %+v %v", m, err)
			}
			if tc.name == "block" && !errors.Is(err, ErrBlockDevice) {
				t.Fatalf("block rejection not distinguishable: %v", err)
			}
		})
	}
}

func TestUnknownClassificationNeverOpensData(t *testing.T) {
	if err := Check(nil); err == nil {
		t.Fatal("missing borrowed input accepted")
	}
	for _, m := range []*fakeMetadata{{}, {statErr: os.ErrPermission}, {statErr: os.ErrNotExist}} {
		if _, err := open("/fixture", from(m)); err == nil || m.dataOpens != 0 || m.closes != 1 {
			t.Fatalf("unknown metadata was treated as safe: %+v %v", m, err)
		}
	}
	if _, err := open("/fixture", from(nil)); err == nil {
		t.Fatal("missing metadata handle accepted")
	}
	if _, err := open("/fixture", func(string) (metadata, error) { return nil, os.ErrPermission }); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("acquisition error lost: %v", err)
	}
}

func TestOpenErrorsRetainCleanupErrors(t *testing.T) {
	acquireErr := errors.New("acquire failed")
	openErr := errors.New("data open failed")
	metadataCloseErr := errors.New("metadata close failed")
	readerCloseErr := errors.New("reader close failed")
	m := &fakeMetadata{closeErr: metadataCloseErr}
	_, err := open("/fixture", func(string) (metadata, error) { return m, acquireErr })
	if !errors.Is(err, acquireErr) || !errors.Is(err, metadataCloseErr) || m.closes != 1 {
		t.Fatalf("acquisition cleanup errors lost: %v", err)
	}
	f := &fakeFile{closeErr: readerCloseErr}
	m = &fakeMetadata{info: regularInfo(t), reader: f, openErr: openErr, closeErr: metadataCloseErr}
	_, err = open("/fixture", from(m))
	if !errors.Is(err, openErr) || !errors.Is(err, metadataCloseErr) || !errors.Is(err, readerCloseErr) || m.closes != 1 || f.closes != 1 {
		t.Fatalf("data-open cleanup errors lost: %v", err)
	}
}

func TestLaterUnavailableMetadataIsNotInputAbsence(t *testing.T) {
	for _, m := range []*fakeMetadata{
		{statErr: os.ErrNotExist},
		{info: regularInfo(t), openErr: os.ErrNotExist},
		{info: regularInfo(t), reader: &fakeFile{statErr: os.ErrNotExist}},
	} {
		_, err := open("/fixture", from(m))
		if !errors.Is(err, ErrUnavailable) || errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			t.Fatalf("later metadata failure misclassified as input absence: %v", err)
		}
	}
}

func regularInfo(t *testing.T) fs.FileInfo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestOpenedInputIdentityAndFailureCleanup(t *testing.T) {
	info := regularInfo(t)
	for _, tc := range []struct {
		name     string
		reader   *fakeFile
		openErr  error
		closeErr error
	}{
		{"permission", nil, os.ErrPermission, nil},
		{"missing-reader", nil, nil, nil},
		{"reader-stat", &fakeFile{statErr: os.ErrPermission}, nil, nil},
		{"reader-missing-info", &fakeFile{}, nil, nil},
		{"changed-inode", &fakeFile{info: regularInfo(t)}, nil, nil},
		{"reader-block", &fakeFile{info: modeInfo{fs.ModeDevice}}, nil, nil},
		{"metadata-close", &fakeFile{info: info}, nil, os.ErrPermission},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &fakeMetadata{info: info, openErr: tc.openErr, closeErr: tc.closeErr}
			if tc.reader != nil {
				m.reader = tc.reader
			}
			if _, err := open("/fixture", from(m)); err == nil || m.closes != 1 || tc.reader != nil && tc.reader.closes != 1 {
				t.Fatalf("failure cleanup: %+v %+v %v", m, tc.reader, err)
			}
		})
	}
}

func TestCheckedReaderIsReturnedWithoutClosingIt(t *testing.T) {
	info := regularInfo(t)
	f := &fakeFile{info: info}
	m := &fakeMetadata{info: info, reader: f}
	got, err := open("/fixture", from(m))
	if err != nil || got != f || m.closes != 1 || f.closes != 0 {
		t.Fatalf("ownership: %+v %+v %v", m, f, err)
	}
}
