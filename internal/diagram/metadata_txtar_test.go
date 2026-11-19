package diagram

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/metadata-*.txtar
var metadataFixtures embed.FS

type diagramFixtureOS struct {
	t      *testing.T
	files  fstest.MapFS
	links  map[string]string
	faults map[string]string
}

func (p diagramFixtureOS) fault(operation, path string) error {
	switch p.faults[operation+":"+path] {
	case "permission":
		return os.ErrPermission
	case "missing":
		return fs.ErrNotExist
	case "other":
		return errors.New("fixture metadata unavailable")
	}
	return nil
}

func (p diagramFixtureOS) ReadDir(path string) ([]fs.DirEntry, error) {
	if err := p.fault("readdir", path); err != nil {
		return nil, err
	}
	return fs.ReadDir(p.files, strings.TrimPrefix(path, "/"))
}

func (p diagramFixtureOS) Open(path string) (fs.File, error) {
	if strings.HasPrefix(path, "/dev/") {
		p.t.Fatalf("device open attempted: %s", path)
	}
	if err := p.fault("open", path); err != nil {
		return nil, err
	}
	return p.files.Open(strings.TrimPrefix(path, "/"))
}

func (p diagramFixtureOS) ReadFile(path string) ([]byte, error) {
	if strings.HasPrefix(path, "/dev/") {
		p.t.Fatalf("device contents read attempted: %s", path)
	}
	if err := p.fault("read", path); err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(p.files, strings.TrimPrefix(path, "/"))
	if errors.Is(err, fs.ErrNotExist) {
		err = fs.ErrNotExist
	}
	return data, err
}

func (p diagramFixtureOS) Readlink(path string) (string, error) {
	if err := p.fault("readlink", path); err != nil {
		return "", err
	}
	if target, ok := p.links[path]; ok {
		return target, nil
	}
	return "", fs.ErrNotExist
}

func (p diagramFixtureOS) Stat(path string) (fs.FileInfo, error) {
	if err := p.fault("stat", path); err != nil {
		return nil, err
	}
	if strings.HasPrefix(path, "/dev/") {
		return sourceInfo(p.t, fs.ModeDevice), nil
	}
	return fs.Stat(p.files, strings.TrimPrefix(path, "/"))
}

func TestMetadataDiagramTXTAR(t *testing.T) {
	files, err := metadataFixtures.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range files {
		t.Run(fixture.Name(), func(t *testing.T) {
			data, err := metadataFixtures.ReadFile("testdata/" + fixture.Name())
			if err != nil {
				t.Fatal(err)
			}
			sections := map[string][]byte{}
			for _, section := range txtar.Parse(data).Files {
				sections[section.Name] = section.Data
			}
			var links map[string]string
			if err := json.Unmarshal(sections["links.json"], &links); err != nil {
				t.Fatal(err)
			}
			var scenarios []struct {
				Name, Reference, Expected string
				Faults                    map[string]string
			}
			if err := json.Unmarshal(sections["scenarios.json"], &scenarios); err != nil {
				t.Fatal(err)
			}
			for _, scenario := range scenarios {
				t.Run(scenario.Name, func(t *testing.T) {
					files := fstest.MapFS{}
					for name, content := range sections {
						if strings.HasPrefix(name, "fs/") {
							files[strings.TrimPrefix(name, "fs/")] = &fstest.MapFile{Data: content}
						}
					}
					for path := range links {
						if strings.HasPrefix(path, "/dev/") {
							files[strings.TrimPrefix(path, "/")] = &fstest.MapFile{Mode: fs.ModeSymlink}
						}
					}
					provider := diagramFixtureOS{t: t, files: files, links: links, faults: scenario.Faults}
					var output bytes.Buffer
					// No test resolver: Print constructs ActualFS and uses canonical
					// inventory enrichment and Resolve on these metadata fixtures.
					err := Print(&output, scenario.Reference, Dependencies{OS: provider, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
						t.Fatal("metadata diagram opened image/device contents")
						return nil, nil
					}})
					if err != nil {
						t.Fatal(err)
					}
					if output.String() != string(sections[scenario.Expected]) {
						t.Fatalf("diagram mismatch\nwant:\n%s\ngot:\n%s", sections[scenario.Expected], output.String())
					}
				})
			}
		})
	}
}

var _ rules.OSProvider = diagramFixtureOS{}
