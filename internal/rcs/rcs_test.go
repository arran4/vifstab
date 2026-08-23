package rcs

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type mockFS struct {
	mkdirAllErr  error
	statErr      error
	statInfo     os.FileInfo
	readFileData []byte
	readFileErr  error
	writeFileErr error
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirAllErr
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	return m.statInfo, m.statErr
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	return m.readFileData, m.readFileErr
}

func (m *mockFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return m.writeFileErr
}

func TestUpdate_Errors(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		fsMock      *mockFS
		expectedErr string
	}{
		{
			name:        "Empty config dir",
			config:      Config{Dir: ""},
			expectedErr: "",
		},
		{
			name: "MkdirAll error",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				mkdirAllErr: errors.New("mkdir failed"),
			},
			expectedErr: "failed to create rcs dir: mkdir failed",
		},
		{
			name: "ReadFile error",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:     nil, // file exists
				readFileErr: errors.New("read failed"),
			},
			expectedErr: "failed to open rcs file: read failed",
		},
		{
			name: "ParseFile error",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:      nil, // file exists
				readFileData: []byte("invalid rcs data"),
			},
			expectedErr: "failed to parse rcs file:",
		},
		{
			name: "WriteFile error new file",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:      os.ErrNotExist, // file doesn't exist
				writeFileErr: errors.New("write failed"),
			},
			expectedErr: "failed to write rcs file: write failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.fsMock != nil {
				err = Update("testfile.txt", []byte("content"), tt.config, tt.fsMock)
			} else {
				err = Update("testfile.txt", []byte("content"), tt.config)
			}

			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Fatalf("expected error containing %q, got %v", tt.expectedErr, err)
				}
			}
		})
	}
}

func TestUpdate_Success(t *testing.T) {
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
@content
@
`

	tests := []struct {
		name   string
		config Config
		fsMock *mockFS
	}{
		{
			name: "New file success",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr: os.ErrNotExist,
			},
		},
		{
			name: "Update existing file success",
			config: Config{Dir: "testdir"},
			fsMock: &mockFS{
				statErr:      nil,
				readFileData: []byte(validRCS),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Update("testfile.txt", []byte("content"), tt.config, tt.fsMock)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
