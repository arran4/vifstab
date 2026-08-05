1.  **Update `internal/rules/builtin/virtual/tracefs/tracefs.go`:**
    *   Define a map of valid mount options `tracefsOptions` based on `tracefs_param_specs` in Linux source:
        *   `gid`: `options.GIDOption{}`
        *   `uid`: `options.UIDOption{}`
        *   `mode`: `options.OctalOption{}` (it is parsed using `fsparam_u32oct` in kernel)
    *   Use `options.MergeOptions` to combine `tracefsOptions` with `options.CommonOptions()`.
    *   Inside the `Validate` function, after checking `fsType`, extract the mount options map from `entry.Fields[3]`.
    *   Pass the combined options map to `options.ValidateOptionsMap` and return its output.

2.  **Create tests for tracefs in `.txtar` format:**
    *   Create a file `internal/rules/builtin/virtual/tracefs/testdata/tracefs.txtar`.
    *   Include valid test cases: `gid=1000,uid=1000,mode=0755` under `-- input/etc/fstab --`.
    *   Include invalid test cases: `mode=abc` (invalid octal), `gid=abc` (invalid group ID), `uid=abc` (invalid user ID) with their expected output in `-- issues --`.
    *   Populate `-- input/etc/passwd --` and `-- input/etc/group --` to mock user/group check.

3.  **Run tests and verify.**
    *   Run `go test ./...`
    *   Check for any errors and fix them.

4.  **Complete pre-commit steps.**
    *   Run the pre-commit checks as directed by `pre_commit_instructions`.

5.  **Submit the change.**
    *   Use the `submit` tool to finalize the implementation.
