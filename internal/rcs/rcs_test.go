package rcs

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-rcs"
)

type mockFS struct {
	mkdirAllErr  error
	statErr      error
	statInfo     os.FileInfo
	readFileData []byte
	readFileErr  error
	writeFileErr error

	writeFilePath  string
	writeFileData  []byte
	writeFilePerm  os.FileMode
	writeFileCalls int
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirAllErr
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	return m.statInfo, m.statErr
}

func (m *mockFS) Lstat(name string) (os.FileInfo, error) {
	if !strings.Contains(filepath.ToSlash(name), "/v1/") {
		return nil, os.ErrNotExist
	}
	info, err := m.Stat(name)
	if info == nil && err == nil {
		return (fstest.MapFS{"history": {Mode: 0600}}).Stat("history")
	}
	return info, err
}
func (m *mockFS) Getwd() (string, error)         { return "/fixture", nil }
func (m *mockFS) Lock(string) (io.Closer, error) { return io.NopCloser(strings.NewReader("")), nil }

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	return m.readFileData, m.readFileErr
}

func (m *mockFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	m.writeFilePath = name
	m.writeFileData = data
	m.writeFilePerm = perm
	m.writeFileCalls++
	return m.writeFileErr
}

func (m *mockFS) SyncDir(name string) error {
	return nil
}

type mockTempFile struct {
	name string
	mfs  *mockFS
}

func (t *mockTempFile) Write(p []byte) (n int, err error) {
	t.mfs.writeFileData = append(t.mfs.writeFileData, p...)
	return len(p), nil
}

func (t *mockTempFile) Close() error {
	t.mfs.writeFileCalls++
	return nil
}

func (t *mockTempFile) Name() string {
	return t.name
}

func (t *mockTempFile) Chmod(mode os.FileMode) error {
	t.mfs.writeFilePerm = mode
	return nil
}

func (t *mockTempFile) Sync() error {
	return nil
}

func (m *mockFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if m.writeFileErr != nil {
		return nil, m.writeFileErr
	}
	name := filepath.Join(dir, strings.ReplaceAll(pattern, "*", "mock"))
	return &mockTempFile{name: name, mfs: m}, nil
}

func (m *mockFS) Rename(oldpath, newpath string) error {
	if m.writeFileErr != nil {
		return m.writeFileErr
	}
	m.writeFilePath = newpath
	return nil
}

func (m *mockFS) Remove(name string) error {
	return nil
}

func TestUpdate_Errors(t *testing.T) {
	errGeneric := errors.New("generic error")
	errPermission := os.ErrPermission

	tests := []struct {
		name        string
		config      Config
		fsMock      *mockFS
		expectedErr error
		expectedStr string
	}{
		{
			name:        "Empty config dir",
			config:      Config{Dir: ""},
			expectedErr: nil,
		},
		{
			name:   "MkdirAll failure",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				mkdirAllErr: errGeneric,
			},
			expectedErr: errGeneric,
			expectedStr: "failed to create rcs dir",
		},
		{
			name:   "Stat failure other than NotExist",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr: errPermission,
			},
			expectedErr: errPermission,
			expectedStr: "failed to stat rcs file",
		},
		{
			name:   "ReadFile failure",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:     nil, // file exists
				readFileErr: errGeneric,
			},
			expectedErr: errGeneric,
			expectedStr: "failed to open rcs file",
		},
		{
			name:   "Invalid RCS parse failure",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:      nil, // file exists
				readFileData: []byte("invalid rcs data"),
			},
			expectedStr: "failed to parse rcs file",
		},
		{
			name:   "Unsupported revision format",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr: nil,
				readFileData: []byte(`head	1;
access;
symbols;
locks; strict;
comment	@# @;


1
date	2023.01.01.00.00.00;	author user;	state Exp;
branches;
next	;


desc
@@


1
log
@Initial revision
@
text
@content
@
`),
			},
			expectedStr: "unsupported revision format: 1",
		},
		{
			name:   "Malformed head parsing",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr: nil,
				readFileData: []byte(`head	1.not-a-number;
access;
symbols;
locks; strict;
comment	@# @;


1.not-a-number
date	2023.01.01.00.00.00;	author user;	state Exp;
branches;
next	;


desc
@@


1.not-a-number
log
@Initial revision
@
text
@content
@
`),
			},
			expectedStr: "failed to parse rcs file",
		},
		{
			name:   "WriteFile failure",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:      os.ErrNotExist,
				writeFileErr: errGeneric,
			},
			expectedErr: errGeneric,
			expectedStr: "failed to create temporary rcs file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.fsMock != nil {
				labelFixture(t, tt.fsMock, "testfile.txt", tt.config.Dir)
				err = Update("testfile.txt", []byte("content"), tt.config, tt.fsMock)
			} else {
				err = Update("testfile.txt", []byte("content"), tt.config)
			}

			if tt.expectedStr == "" && tt.expectedErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error to wrap %v, got %v", tt.expectedErr, err)
			}
			if tt.expectedStr != "" && !strings.Contains(err.Error(), tt.expectedStr) {
				t.Errorf("expected error string to contain %q, got %v", tt.expectedStr, err)
			}

			if tt.fsMock != nil && tt.name == "Stat failure other than NotExist" {
				if tt.fsMock.writeFileCalls != 0 {
					t.Errorf("expected 0 WriteFile calls on Stat error, got %d", tt.fsMock.writeFileCalls)
				}
			}
		})
	}
}

func TestUpdate_NewFileSuccess(t *testing.T) {
	fsMock := &mockFS{
		statErr: os.ErrNotExist,
	}
	config := Config{Dir: "testdir"}
	content := []byte("new file content")

	err := Update("testfile.txt", content, config, fsMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fsMock.writeFileCalls != 1 {
		t.Fatalf("expected TempFile Close to be called exactly once (recording write), got %d", fsMock.writeFileCalls)
	}

	expectedPath, err := HistoryPath("testfile.txt", config.Dir, fsMock)
	if err != nil {
		t.Fatal(err)
	}
	if fsMock.writeFilePath != expectedPath {
		t.Errorf("expected Rename to target path %q, got %q", expectedPath, fsMock.writeFilePath)
	}

	if fsMock.writeFilePerm != 0600 {
		t.Errorf("expected TempFile Chmod permissions 0600, got %v", fsMock.writeFilePerm)
	}

	// Parse the written data
	parsedFile, err := rcs.ParseFile(bytes.NewReader(fsMock.writeFileData))
	if err != nil {
		t.Fatalf("written data is not valid RCS: %v", err)
	}

	if parsedFile.Head != "1.1" {
		t.Errorf("expected Head '1.1', got %q", parsedFile.Head)
	}

	if len(parsedFile.RevisionContents) != 1 {
		t.Fatalf("expected 1 revision content, got %d", len(parsedFile.RevisionContents))
	}

	if parsedFile.RevisionContents[0].Revision != "1.1" {
		t.Errorf("expected revision content '1.1', got %q", parsedFile.RevisionContents[0].Revision)
	}

	if parsedFile.RevisionContents[0].Text != string(content) {
		t.Errorf("expected revision text %q, got %q", string(content), parsedFile.RevisionContents[0].Text)
	}

	if parsedFile.RevisionContents[0].Log != "Initial revision" {
		t.Errorf("expected initial log, got %q", parsedFile.RevisionContents[0].Log)
	}

	if !parsedFile.Strict {
		t.Errorf("expected Strict to be true")
	}
}

func TestUpdate_ExistingFileSuccess(t *testing.T) {
	validRCS := `head	1.1;
access;
symbols;
locks; strict;
comment	@# @;


1.1
date	2023.01.01.00.00.00;	author user;	state Exp;
branches;
next	;


desc
@@


1.1
log
@Initial revision
@
text
@old content
@
`
	fsMock := &mockFS{
		statErr:      nil,
		readFileData: []byte(validRCS),
	}
	config := Config{Dir: "testdir"}
	newContent := []byte("updated content")
	labelFixture(t, fsMock, "testfile.txt", config.Dir)

	err := Update("testfile.txt", newContent, config, fsMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fsMock.writeFileCalls != 1 {
		t.Fatalf("expected TempFile Close to be called exactly once (recording write), got %d", fsMock.writeFileCalls)
	}

	expectedPath, err := HistoryPath("testfile.txt", config.Dir, fsMock)
	if err != nil {
		t.Fatal(err)
	}
	if fsMock.writeFilePath != expectedPath {
		t.Errorf("expected Rename to target path %q, got %q", expectedPath, fsMock.writeFilePath)
	}

	if fsMock.writeFilePerm != 0600 {
		t.Errorf("expected TempFile Chmod permissions 0600, got %v", fsMock.writeFilePerm)
	}

	// Parse the written data
	parsedFile, err := rcs.ParseFile(bytes.NewReader(fsMock.writeFileData))
	if err != nil {
		t.Fatalf("written data is not valid RCS: %v", err)
	}

	if parsedFile.Head != "1.2" {
		t.Errorf("expected Head '1.2', got %q", parsedFile.Head)
	}

	if len(parsedFile.RevisionContents) != 2 {
		t.Fatalf("expected 2 revision contents, got %d", len(parsedFile.RevisionContents))
	}

	// Check 1.2
	if parsedFile.RevisionContents[0].Revision != "1.2" {
		t.Errorf("expected first revision to be '1.2', got %q", parsedFile.RevisionContents[0].Revision)
	}
	if parsedFile.RevisionContents[0].Text != string(newContent) {
		t.Errorf("expected new revision text %q, got %q", string(newContent), parsedFile.RevisionContents[0].Text)
	}
	if parsedFile.RevisionContents[0].Log != "Updated revision" {
		t.Errorf("expected updated log, got %q", parsedFile.RevisionContents[0].Log)
	}

	// Check 1.1 retention
	if parsedFile.RevisionContents[1].Revision != "1.1" {
		t.Errorf("expected second revision to be '1.1', got %q", parsedFile.RevisionContents[1].Revision)
	}

	identity, _, err := historyIdentity("testfile.txt", config.Dir, fsMock)
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := parseProvenance(parsedFile, identity)
	if err != nil {
		t.Fatal(err)
	}
	old, err := restoreRevision(parsedFile, provenance, "1.1")
	if err != nil || string(old) != "old content\n" {
		t.Errorf("expected recoverable original revision, got %q, %v", old, err)
	}
}

// Port valid legacy-shaped unit data to the labelled fixture format only.
// Malformed inputs stay malformed; production never automatically imports them.
func labelFixture(t *testing.T, fs *mockFS, source, dir string) {
	t.Helper()
	f, err := rcs.ParseFile(bytes.NewReader(fs.readFileData))
	if err != nil || len(f.RevisionContents) != 1 {
		return
	}
	identity, _, err := historyIdentity(source, dir, fs)
	if err != nil {
		t.Fatal(err)
	}
	if err := setProvenance(f, identity, f.Head, []byte(f.RevisionContents[0].Text), nil); err != nil {
		t.Fatal(err)
	}
	fs.readFileData = []byte(f.String())
}
