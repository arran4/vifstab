package rcs

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestNewDirectoryEntriesAreSyncedBeforeHistory(t *testing.T) {
	for _, failure := range []string{"", "stat", "mkdir", "sync"} {
		t.Run(failure, func(t *testing.T) {
			var synced []string
			created := false
			info, err := (fstest.MapFS{"existing": &fstest.MapFile{Mode: os.ModeDir | 0700}}).Stat("existing")
			if err != nil {
				t.Fatal(err)
			}
			stat := func(path string) (os.FileInfo, error) {
				if failure == "stat" {
					return nil, os.ErrPermission
				}
				if path == "/existing" {
					return info, nil
				}
				return nil, os.ErrNotExist
			}
			mkdir := func(path string, mode os.FileMode) error {
				if path != "/existing/new/deep/v1" || mode != 0700 {
					t.Fatalf("unexpected creation %s %v", path, mode)
				}
				if failure == "mkdir" {
					return os.ErrPermission
				}
				created = true
				return nil
			}
			sync := func(path string) error {
				if !created {
					t.Fatal("directory sync before creation")
				}
				synced = append(synced, path)
				if failure == "sync" {
					return os.ErrPermission
				}
				return nil
			}
			err = makeDurableDirs("/existing/new/deep/v1", 0700, stat, mkdir, sync)
			if failure != "" {
				if !errors.Is(err, os.ErrPermission) {
					t.Fatalf("failure ignored: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(synced, []string{"/existing", "/existing/new", "/existing/new/deep"}) {
				t.Fatalf("creation durability order: %v", synced)
			}
		})
	}
}
