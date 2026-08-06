1. **Identify the goal**: Make `go.mod` as minimal as possible, primarily by replacing or removing dependencies like `golang.org/x/tools/txtar`.
2. **Current state**:
   - `golang.org/x/tools/txtar` is used in three test files: `internal/rules/builtin/txtar_test.go`, `internal/rules/runner_test.go`, and `cmd/vifstab/main_test.go`.
   - The user suggested replacing deps with something internal. `txtar` format parser is quite small, I can just copy the `txtar` package from `golang.org/x/tools/txtar` to `internal/txtar` or copy the source code of `txtar` directly to `internal/txtar` within the project.
3. **Plan steps**:
   - Create a directory `internal/txtar`.
   - Copy the source code of `txtar` from `golang.org/x/tools` to `internal/txtar`.
   - Update imports in `internal/rules/builtin/txtar_test.go`, `internal/rules/runner_test.go`, and `cmd/vifstab/main_test.go` to use `github.com/arran4/vifstab/internal/txtar`.
   - Run `go mod tidy` to clean up `go.mod`.
   - Run tests to ensure everything still passes.
   - Run pre-commit checks.
   - Submit the change.
