// Package workflow tests the repository's actual event shell and job conditions.
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

func workflowJob(t *testing.T, text, name string) string {
	t.Helper()
	start := strings.Index(text, "\n  "+name+":\n")
	if start < 0 {
		t.Fatalf("missing job %s", name)
	}
	job := text[start+1:]
	if next := regexp.MustCompile(`(?m)^  [a-zA-Z0-9_-]+:\s*$`).FindStringIndex(job[len(name)+5:]); next != nil {
		job = job[:len(name)+5+next[0]]
	}
	return job
}

func expression(t *testing.T, text string) string {
	t.Helper()
	start := strings.Index(text, "${{")
	end := strings.Index(text, "}}")
	if start < 0 || end < start {
		t.Fatal("missing workflow expression")
	}
	return text[start+3 : end]
}

// Evaluate the small expression subset used by these jobs from the actual YAML,
// including GitHub's operand-valued &&/|| and implicit success() admission rule.
// Actionlint separately validates the full workflow schema and expressions.
func evaluate(t *testing.T, raw string, values map[string]any, cancelled, success bool) any {
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
				return !truth(eval(e.X))
			}
		case *ast.BinaryExpr:
			left := eval(e.X)
			switch e.Op {
			case token.LAND:
				if !truth(left) {
					return left
				}
				return eval(e.Y)
			case token.LOR:
				if truth(left) {
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
func truth(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		return false
	}
}

func TestReleaseEventGraph(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	route := workflowJob(t, text, "route")
	release := workflowJob(t, text, "goreleaser")
	scriptStart := strings.Index(route, "        run: |\n")
	if scriptStart < 0 {
		t.Fatal("missing route shell")
	}
	var lines []string
	for _, line := range strings.Split(route[scriptStart+len("        run: |\n"):], "\n") {
		lines = append(lines, strings.TrimPrefix(line, "          "))
	}
	script := strings.Join(lines, "\n")
	condition := expression(t, release)
	// These are intentionally required gates; skipped release preparation is the
	// sole supported skipped dependency for an ordinary v-tag push.
	for _, name := range []string{"route", "go-test", "go-vet", "prepare-release-tag"} {
		if !strings.Contains(strings.Split(release, "\n    if:")[0], name) {
			t.Fatalf("missing release dependency %s", name)
		}
	}
	cases := []struct {
		name, event, ref, mode, action, schedule, prep, test, vet, route string
		cancel, code, publish, monthly, nightly                          bool
	}{
		{name: "main push", event: "push", ref: "refs/heads/main", code: true},
		{name: "v tag, preparation intentionally skipped", event: "push", ref: "refs/tags/v1.2.3", code: true, publish: true},
		{name: "prerelease v tag", event: "push", ref: "refs/tags/v1.2.3-rc.1", code: true, publish: true},
		{name: "test tag only checks", event: "push", ref: "refs/tags/test-123", code: true},
		{name: "ordinary PR", event: "pull_request", action: "opened", code: true},
		{name: "closed PR", event: "pull_request", action: "closed"},
		{name: "published release validates without republishing", event: "release", action: "published", code: true},
		{name: "manual patch", event: "workflow_dispatch", mode: "release-patch", prep: "success", code: true, publish: true},
		{name: "manual major", event: "workflow_dispatch", mode: "release-major", prep: "success", code: true, publish: true},
		{name: "manual minor", event: "workflow_dispatch", mode: "release-minor", prep: "success", code: true, publish: true},
		{name: "manual test snapshot", event: "workflow_dispatch", mode: "release-test", prep: "success", code: true, publish: true},
		{name: "manual rc snapshot", event: "workflow_dispatch", mode: "release-rc", prep: "success", code: true, publish: true},
		{name: "manual alpha snapshot", event: "workflow_dispatch", mode: "release-alpha", prep: "success", code: true, publish: true},
		{name: "manual build", event: "workflow_dispatch", mode: "build", code: true},
		{name: "manual lint", event: "workflow_dispatch", mode: "lint-fix", code: true, nightly: true},
		{name: "monthly", event: "schedule", schedule: "0 19 1 * *", code: true, monthly: true},
		{name: "manual monthly", event: "workflow_dispatch", mode: "monthly-maintenance", code: true, monthly: true},
		{name: "unknown event", event: "unknown"},
		{name: "tag test failure", event: "push", ref: "refs/tags/v1.2.3", test: "failure", code: true},
		{name: "tag vet failure", event: "push", ref: "refs/tags/v1.2.3", vet: "failure", code: true},
		{name: "tag lint prevents tests", event: "push", ref: "refs/tags/v1.2.3", test: "skipped", code: true},
		{name: "route failure", event: "push", ref: "refs/tags/v1.2.3", route: "failure", code: true},
		{name: "cancelled tag", event: "push", ref: "refs/tags/v1.2.3", cancel: true, code: true},
		{name: "failed preparation", event: "workflow_dispatch", mode: "release-patch", prep: "failure", code: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]any{"github.event_name": tc.event, "github.ref": tc.ref, "inputs.mode": tc.mode, "github.event.action": tc.action, "github.event.schedule": tc.schedule}
			run := script
			for field, value := range values {
				run = strings.ReplaceAll(run, "${{ "+field+" }}", value.(string))
			}
			output := filepath.Join(t.TempDir(), "outputs")
			command := exec.Command("bash", "-euo", "pipefail", "-c", run)
			command.Env = append(os.Environ(), "GITHUB_OUTPUT="+output,
				"VIFSTAB_EVENT_NAME="+tc.event, "VIFSTAB_EVENT_REF="+tc.ref, "VIFSTAB_EVENT_ACTION="+tc.action, "VIFSTAB_EVENT_SCHEDULE="+tc.schedule, "VIFSTAB_PIPELINE_MODE="+tc.mode)
			if out, err := command.CombinedOutput(); err != nil {
				t.Fatalf("route shell: %v: %s", err, out)
			}
			b, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			outputs := map[string]string{}
			for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
				k, v, ok := strings.Cut(line, "=")
				if !ok {
					t.Fatal(line)
				}
				outputs[k] = v
			}
			if outputs["is_monthly"] != fmt.Sprint(tc.monthly) || outputs["is_nightly"] != fmt.Sprint(tc.nightly) {
				t.Fatalf("maintenance routing changed: %s", b)
			}
			if (outputs["run_code_checks"] == "true") != tc.code {
				t.Fatalf("routing changed: %s", b)
			}
			prep := tc.prep
			if prep == "" {
				prep = "skipped"
			}
			test, vet, routeStatus := tc.test, tc.vet, tc.route
			if test == "" {
				test = "success"
				if !tc.code {
					test = "skipped"
				}
			}
			if vet == "" {
				vet = "success"
				if !tc.code {
					vet = "skipped"
				}
			}
			if routeStatus == "" {
				routeStatus = "success"
			}
			values["needs.route.result"] = routeStatus
			values["needs.go-test.result"] = test
			values["needs.go-vet.result"] = vet
			values["needs.prepare-release-tag.result"] = prep
			values["needs.route.outputs.run_release"] = outputs["run_release"]
			success := routeStatus == "success" && test == "success" && vet == "success" && prep == "success"
			admitted := truth(evaluate(t, condition, values, tc.cancel, success))
			if !regexp.MustCompile(`\b(always|cancelled|success|failure)\s*\(`).MatchString(condition) {
				admitted = admitted && success
			}
			if admitted {
				tagLine := release[strings.Index(release, "GORELEASER_CURRENT_TAG:"):]
				tag := "v2.3.4"
				if tc.event == "push" {
					tag = strings.TrimPrefix(tc.ref, "refs/tags/")
				}
				values["needs.prepare-release-tag.outputs.release_tag"] = ""
				if tc.event == "workflow_dispatch" {
					values["needs.prepare-release-tag.outputs.release_tag"] = "v2.3.4"
				}
				values["github.ref_name"] = strings.TrimPrefix(tc.ref, "refs/tags/")
				if got := evaluate(t, expression(t, tagLine), values, tc.cancel, success); got != tag {
					t.Fatalf("release tag %v want %v", got, tag)
				}
			}
			if admitted != tc.publish {
				t.Fatalf("release admitted=%v want %v; route=%s condition=%s", admitted, tc.publish, b, condition)
			}
			step := release[strings.Index(release, "      - name: Run GoReleaser"):]
			snapshot := truth(evaluate(t, expression(t, step), values, tc.cancel, success))
			wantSnapshot := tc.event == "workflow_dispatch" && (tc.mode == "release-test" || tc.mode == "release-rc" || tc.mode == "release-alpha")
			if snapshot != wantSnapshot {
				t.Fatalf("snapshot selection=%v want %v", snapshot, wantSnapshot)
			}
			if tc.event == "pull_request" && outputs["run_pr_meta_checks"] != fmt.Sprint(tc.action != "closed") {
				t.Fatal("PR metadata routing changed")
			}
			if tc.action == "closed" && outputs["run_cleanup"] != "true" {
				t.Fatal("closed PR cleanup changed")
			}
		})
	}
}
