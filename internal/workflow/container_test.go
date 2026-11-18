package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestContainerReleaseConfiguration(t *testing.T) {
	root := filepath.Join("..", "..")
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	config := read(".goreleaser.yaml")
	tags := securitySections(config, 0)["dockers_v2"]
	for _, required := range []string{"ghcr.io/arran4/vifstab", "linux/amd64", "linux/arm64", "THIRD_PARTY_NOTICES.md", "org.opencontainers.image.source"} {
		if !strings.Contains(tags, required) {
			t.Errorf("missing container configuration %s", required)
		}
	}
	for _, tc := range []struct {
		tag, prerelease string
		snapshot bool
		want string
	}{
		{"v1.2.3", "", false, "v1.2.3 latest"},
		{"v1.2.3-rc.1", "rc.1", false, "v1.2.3-rc.1"},
		{"v1.2.3", "", true, "snapshot"},
	} {
		var got []string
		for _, line := range strings.Split(tags, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "- '{{") {
				continue
			}
			tmpl, err := template.New("tag").Parse(strings.TrimSuffix(strings.TrimPrefix(line, "- '"), "'"))
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := tmpl.Execute(&out, struct {
				Tag, Prerelease string
				IsSnapshot bool
			}{tc.tag, tc.prerelease, tc.snapshot}); err != nil {
				t.Fatal(err)
			}
			if out.Len() > 0 {
				got = append(got, out.String())
			}
		}
		if strings.Join(got, " ") != tc.want {
			t.Errorf("tags %v, want %s", got, tc.want)
		}
	}
	workflow := read(".github/workflows/container-snapshot.yml")
	if strings.Contains(workflow, "packages: write") || strings.Contains(workflow, "docker/login-action") || strings.Contains(workflow, "secrets.") || strings.Contains(workflow, "pull_request_target") {
		t.Fatal("snapshot workflow must not have publication credentials")
	}
	if !strings.Contains(workflow, "release --snapshot --clean") || !strings.Contains(workflow, "scripts/test-container-images.sh") {
		t.Fatal("snapshot workflow must build and test actual images")
	}
	ci := read(".github/workflows/ci.yml")
	snapshot := workflowJob(t, ci, "goreleaser-snapshot")
	if strings.Contains(snapshot, "packages: write") || strings.Contains(snapshot, "docker/login-action") || strings.Contains(snapshot, "secrets.") {
		t.Fatal("manual snapshot must not have publication credentials")
	}
	release := workflowJob(t, ci, "goreleaser")
	if !strings.Contains(release, "packages: write") || !strings.Contains(release, "registry: ghcr.io") {
		t.Fatal("publishing job requires scoped GHCR authentication")
	}
}
