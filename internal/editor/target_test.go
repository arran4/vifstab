package editor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/config"
	"golang.org/x/tools/txtar"
)

// Each case builds only temporary regular files and symlinks. The same injected
// osFS resolver used for real saves resolves these fixtures; no host fstab is read.
func TestSaveSymlinkFixtures(t *testing.T) {
	paths, err := filepath.Glob("testdata/symlink_save/*.txtar")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			sections := map[string]string{}
			for _, f := range txtar.Parse(data).Files {
				sections[f.Name] = strings.ReplaceAll(string(f.Data), "${ROOT}", root)
			}
			for name, data := range sections {
				if !strings.HasPrefix(name, "file/") {
					continue
				}
				dest := filepath.Join(root, strings.TrimPrefix(name, "file/"))
				if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dest, []byte(data), 0640); err != nil {
					t.Fatal(err)
				}
			}
			for name, data := range sections {
				if !strings.HasPrefix(name, "link/") {
					continue
				}
				dest := filepath.Join(root, strings.TrimPrefix(name, "link/"))
				if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(strings.TrimSpace(data), dest); err != nil {
					t.Fatal(err)
				}
			}
			// Do not clean: symlink/.. deliberately differs from lexical cleaning.
			dest := root + "/" + strings.TrimSpace(sections["path"])
			res := SaveAndVersion(config.Config{FilePath: dest}, []byte(sections["new"]), osFS{})
			wantErr := strings.TrimSpace(sections["error"])
			if wantErr == "" {
				if res.Err != nil || !res.Saved {
					t.Fatalf("save: %+v", res)
				}
			} else if res.Err == nil || !strings.Contains(res.Err.Error(), wantErr) || res.Saved {
				t.Fatalf("want unsaved error %q, got %+v", wantErr, res)
			}
			for name, original := range sections {
				switch {
				case strings.HasPrefix(name, "link/"):
					p := filepath.Join(root, strings.TrimPrefix(name, "link/"))
					got, err := os.Readlink(p)
					if err != nil || got != strings.TrimSpace(original) {
						t.Errorf("link changed %s: %q, %v", name, got, err)
					}
				case strings.HasPrefix(name, "file/"):
					rel := strings.TrimPrefix(name, "file/")
					want := original
					if updated, ok := sections["expected/"+rel]; ok {
						want = updated
					}
					got, err := os.ReadFile(filepath.Join(root, rel))
					if err != nil || string(got) != want {
						t.Errorf("file %s = %q (%v), want %q", rel, got, err, want)
					}
					info, err := os.Stat(filepath.Join(root, rel))
					if err != nil || info.Mode().Perm() != 0640 {
						t.Errorf("mode changed for %s: %v, %v", rel, info, err)
					}
				case strings.HasPrefix(name, "expected/"):
					rel := strings.TrimPrefix(name, "expected/")
					if _, ok := sections["file/"+rel]; ok {
						continue
					}
					got, err := os.ReadFile(filepath.Join(root, rel))
					if err != nil || string(got) != original {
						t.Errorf("new file %s = %q (%v), want %q", rel, got, err, original)
					}
				}
			}
			err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if strings.HasPrefix(d.Name(), "vifstab-tmp-") {
					t.Errorf("temporary destination leaked: %s", path)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

type targetFaultFS struct {
	FS
	lstatErr, evalErr, statErr error
	renameCalled               bool
	beforeClose                func()
}

func (f *targetFaultFS) Lstat(name string) (os.FileInfo, error) {
	if f.lstatErr != nil {
		return nil, f.lstatErr
	}
	return f.FS.Lstat(name)
}
func (f *targetFaultFS) EvalSymlinks(name string) (string, error) {
	if f.evalErr != nil {
		return "", f.evalErr
	}
	return f.FS.EvalSymlinks(name)
}
func (f *targetFaultFS) Stat(name string) (os.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.FS.Stat(name)
}
func (f *targetFaultFS) Rename(from, to string) error {
	f.renameCalled = true
	return f.FS.Rename(from, to)
}
func (f *targetFaultFS) CreateTemp(dir, pattern string) (TempFile, error) {
	tmp, err := f.FS.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &targetCloseFile{TempFile: tmp, beforeClose: f.beforeClose}, nil
}
func (f *targetFaultFS) Chown(name string, uid, gid int) error {
	if ch, ok := f.FS.(Chowner); ok {
		return ch.Chown(name, uid, gid)
	}
	return errors.New("chown unsupported")
}
func (f *targetFaultFS) Getxattr(path, name string) ([]byte, error) {
	if xh, ok := f.FS.(XattrHandler); ok {
		return xh.Getxattr(path, name)
	}
	return nil, errors.New("xattr unsupported")
}
func (f *targetFaultFS) Setxattr(path, name string, value []byte) error {
	if xh, ok := f.FS.(XattrHandler); ok {
		return xh.Setxattr(path, name, value)
	}
	return errors.New("xattr unsupported")
}
func (f *targetFaultFS) Listxattr(path string) ([]string, error) {
	if xh, ok := f.FS.(XattrHandler); ok {
		return xh.Listxattr(path)
	}
	return nil, errors.New("xattr unsupported")
}
func (f *targetFaultFS) Removexattr(path, name string) error {
	if xr, ok := f.FS.(XattrRemover); ok {
		return xr.Removexattr(path, name)
	}
	return errors.New("xattr remove unsupported")
}

type targetCloseFile struct {
	TempFile
	beforeClose func()
}

func (f *targetCloseFile) Chown(uid, gid int) error {
	if ch, ok := f.TempFile.(TempFileChowner); ok {
		return ch.Chown(uid, gid)
	}
	return errors.New("chown unsupported")
}

func (f *targetCloseFile) Close() error {
	if f.beforeClose != nil {
		f.beforeClose()
	}
	return f.TempFile.Close()
}

func TestSaveTargetMetadataErrors(t *testing.T) {
	for _, op := range []string{"lstat", "resolve", "stat"} {
		t.Run(op, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "fstab")
			if err := os.WriteFile(dest, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			f := &targetFaultFS{FS: osFS{}}
			switch op {
			case "lstat":
				f.lstatErr = os.ErrPermission
			case "resolve":
				f.evalErr = os.ErrPermission
			case "stat":
				f.statErr = os.ErrPermission
			}
			res := SaveAndVersion(config.Config{FilePath: dest}, []byte("edited"), f)
			if !errors.Is(res.Err, os.ErrPermission) || res.Saved || f.renameCalled {
				t.Fatalf("unsafe result: %+v rename=%v", res, f.renameCalled)
			}
			data, err := os.ReadFile(dest)
			if err != nil || string(data) != "original" {
				t.Fatalf("destination changed: %q %v", data, err)
			}
		})
	}
}

func TestSaveDetectsSymlinkRetargetingBeforeRename(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	for _, name := range []string{"first", "second"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("first", link); err != nil {
		t.Fatal(err)
	}
	f := &targetFaultFS{FS: osFS{}, beforeClose: func() {
		if err := os.Remove(link); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("second", link); err != nil {
			t.Fatal(err)
		}
	}}
	res := SaveAndVersion(config.Config{FilePath: link}, []byte("edited"), f)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "target changed") || res.Saved || f.renameCalled {
		t.Fatalf("unsafe result: %+v rename=%v", res, f.renameCalled)
	}
	for _, name := range []string{"first", "second"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(data) != name {
			t.Fatalf("%s changed: %q %v", name, data, err)
		}
	}
}
