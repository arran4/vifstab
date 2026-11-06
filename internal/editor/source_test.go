package editor

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
)

// The existing systemd workflow fixture uses a virtual /etc/fstab. Source reads
// must use the injected FS as saves do, never open the host's similarly named file.
func TestEditSourceUsesInjectedFilesystem(t *testing.T) {
	const path = "/virtual-only-vifstab-source/fstab"
	const content = "# source from fixture, never from host\n"
	fs := &mockFS{mfs: fstest.MapFS{"virtual-only-vifstab-source/fstab": {Data: []byte(content), Mode: 0600}}}
	stop := errors.New("stop after checking editor input")
	called := false
	err := EditAndLint(config.Config{FilePath: path}, &mockAPI{}, Dependencies{FS: fs, Editor: func(path string) error {
		called = true
		got, err := os.ReadFile(path)
		if err != nil || string(got) != content {
			t.Fatalf("editor source = %q, %v", got, err)
		}
		return stop
	}})
	if !called || !errors.Is(err, stop) {
		t.Fatalf("injected source not supplied: editor=%v, err=%v", called, err)
	}
}
