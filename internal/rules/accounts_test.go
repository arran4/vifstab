package rules_test

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
)

func TestAccountFilesArePositiveObservationsNotCompleteInventories(t *testing.T) {
	validUser := "example:x:1000:1000::/fixture/home:/bin/false\n"
	validGroup := "example:x:1000:\n"
	for _, tc := range []struct {
		name   string
		passwd string
		group  string
		user   rules.ExistState
		gid    rules.ExistState
	}{
		{"valid-match", validUser, validGroup, rules.Exists, rules.Exists},
		{"valid-without-final-lf", strings.TrimSuffix(validUser, "\n"), strings.TrimSuffix(validGroup, "\n"), rules.Exists, rules.Exists},
		{"file-only-missing", "root:x:0:0::/fixture/root:/bin/false\n", "root:x:0:\n", rules.Unknown, rules.Unknown},
		{"empty-files", "", "", rules.Unknown, rules.Unknown},
		{"truncated-match", "example:x:1000\n", "example:x:1000\n", rules.Unknown, rules.Unknown},
		{"extra-fields", strings.TrimSpace(validUser) + ":extra\n", strings.TrimSpace(validGroup) + ":extra\n", rules.Unknown, rules.Unknown},
		{"invalid-primary-gid", "example:x:1000:bad::/fixture/home:/bin/false\n", "example:x:bad:\n", rules.Unknown, rules.Unknown},
		{"comment-is-not-account", "#" + validUser, "#" + validGroup, rules.Unknown, rules.Unknown},
		{"compat-directive", "+" + validUser, "-" + validGroup, rules.Unknown, rules.Unknown},
		{"nul-name", "bad\x00:x:1000:1000::/:/bin/false\n", "bad\x00:x:1000:\n", rules.Unknown, rules.Unknown},
		{"empty-name", ":x:1000:1000::/:/bin/false\n", ":x:1000:\n", rules.Unknown, rules.Unknown},
		{"valid-observation-with-other-corruption", "broken\n" + validUser, validGroup + "broken\n", rules.Exists, rules.Exists},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := fstest.MapFS{"etc/passwd": {Data: []byte(tc.passwd)}, "etc/group": {Data: []byte(tc.group)}}
			actual := rules.ActualFS{OS: mockErrorOS{openFunc: func(name string) (fs.File, error) {
				return files.Open(strings.TrimPrefix(name, "/"))
			}}}
			for _, provider := range []struct {
				name string
				api  rules.API
			}{{"ActualFS", actual}, {"CacheFS", rules.NewCacheFS(rules.NewCache(), files)}} {
				t.Run(provider.name, func(t *testing.T) {
					if got := provider.api.CheckUserExists(1000); got != tc.user {
						t.Errorf("user = %v, want %v", got, tc.user)
					}
					if got := provider.api.CheckGroupExists(1000); got != tc.gid {
						t.Errorf("group = %v, want %v", got, tc.gid)
					}
				})
			}
		})
	}
}

type accountReadErrorFile struct {
	fs.File
	err error
}

func (f accountReadErrorFile) Read(data []byte) (int, error) {
	n, err := f.File.Read(data)
	if err == io.EOF {
		return 0, f.err
	}
	return n, err
}

type accountReadErrorFS struct {
	fs.FS
	err error
}

func (f accountReadErrorFS) Open(path string) (fs.File, error) {
	file, err := f.FS.Open(path)
	if err != nil {
		return nil, err
	}
	return accountReadErrorFile{File: file, err: f.err}, nil
}

func TestAccountReadFaultDoesNotCreateAbsence(t *testing.T) {
	files := fstest.MapFS{"etc/passwd": {Data: []byte("example:x:1000:1000::/:/bin/false\n")}, "etc/group": {Data: []byte("example:x:1000:\n")}}
	for _, failure := range []struct {
		err  error
		want rules.ExistState
	}{{errors.New("fixture I/O unavailable"), rules.Unknown}, {fs.ErrPermission, rules.PermissionDenied}} {
		fsys := accountReadErrorFS{FS: files, err: failure.err}
		actual := rules.ActualFS{OS: mockErrorOS{openFunc: func(path string) (fs.File, error) {
			return fsys.Open(strings.TrimPrefix(path, "/"))
		}}}
		for _, api := range []rules.API{actual, rules.NewCacheFS(rules.NewCache(), fsys)} {
			if got := api.CheckUserExists(2000); got != failure.want {
				t.Errorf("unmatched user after read error = %v, want %v", got, failure.want)
			}
			if got := api.CheckGroupExists(2000); got != failure.want {
				t.Errorf("unmatched group after read error = %v, want %v", got, failure.want)
			}
			if api.CheckUserExists(1000) != rules.Exists || api.CheckGroupExists(1000) != rules.Exists {
				t.Error("valid positive observation before the fault was lost")
			}
		}
	}
}

func TestAccountReadFaultDoesNotCompletePartialRecord(t *testing.T) {
	files := fstest.MapFS{
		"etc/passwd": {Data: []byte("example:x:1000:1000::/fixture/home:/bin/par")},
		"etc/group":  {Data: []byte("example:x:1000:partial-mem")},
	}
	fsys := accountReadErrorFS{FS: files, err: errors.New("fixture truncated read")}
	actual := rules.ActualFS{OS: mockErrorOS{openFunc: func(path string) (fs.File, error) {
		return fsys.Open(strings.TrimPrefix(path, "/"))
	}}}
	for _, api := range []rules.API{actual, rules.NewCacheFS(rules.NewCache(), fsys)} {
		if api.CheckUserExists(1000) != rules.Unknown || api.CheckGroupExists(1000) != rules.Unknown {
			t.Error("I/O failure completed a partial account record")
		}
	}
}

func TestExplicitAccountObservationsRemainAuthoritative(t *testing.T) {
	for _, state := range []rules.ExistState{rules.Exists, rules.DoesNotExist, rules.PermissionDenied, rules.Unknown} {
		cache := rules.NewCache()
		cache.Set("user_exists:1000", state)
		cache.Set("group_exists:1000", state)
		api := rules.NewCacheFS(cache, fstest.MapFS{})
		if got := api.CheckUserExists(1000); got != state {
			t.Errorf("explicit user observation = %v, want %v", got, state)
		}
		if got := api.CheckGroupExists(1000); got != state {
			t.Errorf("explicit group observation = %v, want %v", got, state)
		}
	}
	for _, state := range []bool{false, true} {
		cache := rules.NewCache()
		cache.Set("user_exists:1000", state)
		cache.Set("group_exists:1000", state)
		api := rules.NewCacheFS(cache, fstest.MapFS{})
		want := rules.DoesNotExist
		if state {
			want = rules.Exists
		}
		if api.CheckUserExists(1000) != want || api.CheckGroupExists(1000) != want {
			t.Fatal("legacy explicit boolean observation changed")
		}
	}
}
