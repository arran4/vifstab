package workflow

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Evaluate the small expression subset used by these jobs from the actual YAML,
// including GitHub's operand-valued &&/||. Event checks give dependencies success.
// Actionlint separately validates the full workflow schema and expressions.
func securityEvaluate(t *testing.T, raw string, values map[string]any, cancelled, success bool) any {
	t.Helper()
	raw = regexp.MustCompile(`'[^']*'`).ReplaceAllStringFunc(raw, func(s string) string { return strconv.Quote(s[1 : len(s)-1]) })
	raw = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_-]*(\.[a-zA-Z_][a-zA-Z0-9_-]*)+`).ReplaceAllStringFunc(raw, func(s string) string {
		value, ok := values[s]
		if !ok {
			t.Fatalf("missing fixture value %s", s)
		}
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	})
	node, err := parser.ParseExpr(strings.TrimSpace(raw))
	if err != nil {
		t.Fatalf("parse condition: %v\n%s", err, raw)
	}
	var eval func(ast.Expr) any
	eval = func(expr ast.Expr) any {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			return eval(e.X)
		case *ast.BasicLit:
			s, err := strconv.Unquote(e.Value)
			if err != nil {
				t.Fatal(err)
			}
			return s
		case *ast.Ident:
			if e.Name == "true" {
				return true
			}
			if e.Name == "false" {
				return false
			}
		case *ast.UnaryExpr:
			if e.Op == token.NOT {
				return !securityTruth(eval(e.X))
			}
		case *ast.BinaryExpr:
			left := eval(e.X)
			switch e.Op {
			case token.LAND:
				if !securityTruth(left) {
					return left
				}
				return eval(e.Y)
			case token.LOR:
				if securityTruth(left) {
					return left
				}
				return eval(e.Y)
			case token.EQL:
				return left == eval(e.Y)
			case token.NEQ:
				return left != eval(e.Y)
			}
		case *ast.CallExpr:
			name, ok := e.Fun.(*ast.Ident)
			if !ok {
				t.Fatal("unsupported condition call")
			}
			switch name.Name {
			case "cancelled":
				return cancelled
			case "success":
				return success
			case "failure":
				return !success
			case "always":
				return true
			case "startsWith":
				return strings.HasPrefix(eval(e.Args[0]).(string), eval(e.Args[1]).(string))
			}
		}
		t.Fatalf("unsupported condition node %T", expr)
		return nil
	}
	return eval(node)
}
func securityTruth(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		return false
	}
}

// securitySections reads only the workflow's block-mapping layout. Unexpected
// shorthand fails these policy checks; Actionlint checks complete YAML semantics.
func securitySections(text string, indent int) map[string]string {
	pattern := regexp.MustCompile(fmt.Sprintf(`(?m)^%s([a-zA-Z][a-zA-Z0-9_-]*):[^\n]*(?:\n|$)`, strings.Repeat(" ", indent)))
	locs := pattern.FindAllStringSubmatchIndex(text, -1)
	sections := map[string]string{}
	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		sections[text[loc[2]:loc[3]]] = text[loc[1]:end]
	}
	return sections
}
func securityPermissions(t *testing.T, block string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.Contains(v, ":") {
			t.Fatalf("unsupported permissions %q", line)
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}
func securityScript(t *testing.T, text, marker string) string {
	t.Helper()
	start := strings.Index(text, marker)
	if start < 0 {
		t.Fatalf("missing shell step %s", marker)
	}
	text = text[start+len(marker):]
	start = strings.Index(text, "        run: |\n")
	if start < 0 {
		t.Fatalf("missing multiline script %s", marker)
	}
	var lines []string
	for _, line := range strings.Split(text[start+len("        run: |\n"):], "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "          ") {
			break
		}
		lines = append(lines, strings.TrimPrefix(line, "          "))
	}
	return strings.Join(lines, "\n")
}

func securityJobCondition(t *testing.T, job string) string {
	t.Helper()
	start := strings.Index(job, "\n    if:")
	if start < 0 {
		t.Fatal("missing job condition")
	}
	lines := strings.Split(job[start+len("\n    if:"):], "\n")
	value := strings.TrimSpace(lines[0])
	if value == ">-" || value == ">" || value == "|" || value == "|-" {
		var parts []string
		for _, line := range lines[1:] {
			if !strings.HasPrefix(line, "      ") {
				break
			}
			parts = append(parts, strings.TrimSpace(line))
		}
		value = strings.Join(parts, " ")
	}
	if !strings.HasPrefix(value, "${{") || !strings.HasSuffix(value, "}}") {
		t.Fatalf("unsupported job condition %q", value)
	}
	return strings.TrimSpace(value[3 : len(value)-2])
}
func TestPublicWorkflowPermissions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	root := securitySections(text, 0)
	defaults := securityPermissions(t, root["permissions"])
	if len(defaults) != 1 || defaults["contents"] != "read" {
		t.Fatalf("unsafe default permissions: %v", defaults)
	}
	if strings.Contains(text, "pull_request_target") || !strings.Contains(root["on"], "  pull_request:") {
		t.Fatal("untrusted checks must remain pull_request")
	}
	jobs := securitySections(root["jobs"], 2)
	if len(jobs) < 6 {
		t.Fatalf("unexpected job layout: %d", len(jobs))
	}
	for name, job := range jobs {
		fields := securitySections(job, 4)
		permissions := defaults
		if block, ok := fields["permissions"]; ok {
			permissions = securityPermissions(t, block)
		}
		writers := map[string]string{}
		for scope, value := range permissions {
			if value == "write" {
				writers[scope] = value
			} else if scope != "contents" || value != "read" {
				t.Errorf("unreviewed token scope %s: %s=%s", name, scope, value)
			}
		}
		switch name {
		case "go-fmt-pr":
			if len(writers) != 2 || writers["contents"] != "write" || writers["pull-requests"] != "write" {
				t.Errorf("formatting scopes: %v", permissions)
			}
		case "goreleaser":
			if len(writers) != 2 || writers["contents"] != "write" || writers["packages"] != "write" {
				t.Errorf("release scopes: %v", permissions)
			}
		default:
			if len(writers) != 0 {
				t.Errorf("untrusted/check job %s gains write: %v", name, writers)
			}
		}
		lines := strings.Split(job, "\n")
		for i, line := range lines {
			if strings.Contains(line, "uses:") {
				if !regexp.MustCompile(`uses: [A-Za-z0-9_./-]+@[0-9a-f]{40}(\s+#.*)?$`).MatchString(strings.TrimSpace(line)) {
					t.Errorf("unreviewed action reference %s", line)
				}
			}
			if strings.Contains(line, "uses: actions/checkout@") {
				got := ""
				for j := i + 1; j < len(lines) && !strings.HasPrefix(lines[j], "      - "); j++ {
					if strings.Contains(lines[j], "persist-credentials:") {
						got = strings.TrimSpace(strings.SplitN(lines[j], ":", 2)[1])
						break
					}
				}
				want := "false"
				if name == "go-fmt-pr" {
					want = "true"
				}
				if got != want {
					t.Errorf("%s checkout credentials = %q, want %q", name, got, want)
				}
			}
			if strings.HasPrefix(line, "        run:") {
				if strings.Contains(line, "${{") {
					t.Errorf("expression interpolated into shell: %s", line)
				}
				for j := i + 1; j < len(lines) && (strings.TrimSpace(lines[j]) == "" || strings.HasPrefix(lines[j], "          ")); j++ {
					if strings.Contains(lines[j], "${{") {
						t.Errorf("expression interpolated into shell: %s", lines[j])
					}
				}
			}
		}
	}
	// Read the actual conditions with successful prerequisites (and intentionally
	// skipped tag preparation for a tag push). These tests check the event gate;
	// the independent release graph tests cover dependency-status admission.
	cases := []struct {
		event, ref, mode       string
		allow, format, publish bool
	}{
		{"pull_request", "refs/pull/1/merge", "release-patch", true, false, false},
		{"pull_request", "refs/heads/main", "lint-fix", true, false, false},
		{"push", "refs/heads/main", "lint-fix", true, false, false},
		{"schedule", "refs/heads/main", "lint-fix", true, false, false},
		{"release", "refs/tags/v1.2.3", "release-patch", true, false, false},
		{"push", "refs/tags/v1.2.3", "", false, false, true},
		{"workflow_dispatch", "refs/heads/main", "lint-fix", true, true, false},
		{"workflow_dispatch", "refs/heads/main", "lint-fix", false, false, false},
		{"workflow_dispatch", "refs/heads/main", "build", true, false, false},
		{"workflow_dispatch", "refs/heads/main", "release-patch", false, false, true},
		{"workflow_dispatch", "refs/heads/feature", "release-patch", false, false, false},
		{"workflow_dispatch", "refs/heads/main", "release-test", false, false, false},
	}
	for _, tc := range cases {
		for _, name := range []string{"go-fmt-pr", "goreleaser"} {
			condition := securityJobCondition(t, jobs[name])
			values := map[string]any{"github.event_name": tc.event, "github.ref": tc.ref, "inputs.mode": tc.mode, "inputs.allow_prs": tc.allow, "needs.route.outputs.run_release": "true", "needs.route.result": "success", "needs.go-test.result": "success", "needs.go-vet.result": "success", "needs.prepare-release-tag.result": "success"}
			if tc.event == "push" && strings.HasPrefix(tc.ref, "refs/tags/v") {
				values["needs.prepare-release-tag.result"] = "skipped"
			}
			got := securityTruth(securityEvaluate(t, condition, values, false, true))
			want := tc.publish
			if name == "go-fmt-pr" {
				want = tc.format
			}
			if got != want {
				t.Errorf("%s event=%s mode=%s allow=%v admitted=%v want=%v", name, tc.event, tc.mode, tc.allow, got, want)
			}
		}
	}
}
func TestDispatchInputsRemainShellData(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	jobs := securitySections(securitySections(string(data), 0)["jobs"], 2)
	root := t.TempDir()
	marker := filepath.Join(root, "injected")
	hostile := "$(touch " + marker + ")"
	route := securityScript(t, jobs["route"], "      - id: route\n")
	cmd := exec.Command("bash", "-euo", "pipefail", "-c", route)
	cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+filepath.Join(root, "outputs"), "VIFSTAB_EVENT_NAME=workflow_dispatch", "VIFSTAB_EVENT_ACTION=", "VIFSTAB_EVENT_SCHEDULE=", "VIFSTAB_PIPELINE_MODE="+hostile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("route: %v %s", err, out)
	}
	// Stub git: no global config, network or tags are touched by this test.
	if err := os.WriteFile(filepath.Join(root, "git"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	prep := securityScript(t, jobs["prepare-release-tag"], "      - id: tag\n")
	cmd = exec.Command("bash", "-euo", "pipefail", "-c", prep)
	cmd.Env = append(os.Environ(), "PATH="+root+":"+os.Getenv("PATH"), "VIFSTAB_PIPELINE_MODE=release-patch", "VIFSTAB_RELEASE_OVERRIDE=1.2.3"+hostile, "GITHUB_OUTPUT="+filepath.Join(root, "tag-outputs"), "GIT_CONFIG_GLOBAL="+filepath.Join(root, "gitconfig"))
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "Invalid tag format") {
		t.Fatalf("invalid override not refused: %v %s", err, out)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("dispatch data executed: %v", err)
	}
}
