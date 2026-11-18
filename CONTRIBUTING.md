# Contributing

Start with the source and fixtures for the rule you want to improve. A useful
report shows the exact input, expected diagnostic, actual diagnostic and tested
revision. Linux is the current target. Linux 7-series coverage is a development
goal, and BSD support is future work; a fixture is evidence for its case, not a
guarantee for every kernel, filesystem or boot configuration.

Read [AGENTS.md](AGENTS.md) for the safety and architecture constraints. Validation
must use the existing injectable metadata/provider interfaces. Tests must not
mount filesystems, inspect block-device contents, execute system discovery tools,
change privileges or read the test host's real fstab, mount tables or EFI state.
Missing or unreadable observations mean unknown, not proof of absence.

## Run representative tests

Use Go 1.25 or a newer compatible, patched toolchain. From the checkout:

```sh
go test ./internal/rules/builtin -run 'TestAllTxtarRules/local/ext4/testdata/ext4_journal' -v
go test ./internal/uefi/...
go test ./internal/editor/...
go test ./...
go vet ./...
git diff --check
```

The EXT4 selection runs committed TXTAR examples with injected kernel versions.
The UEFI tests construct binary fixtures; they do not need firmware access.
Format changed Go files with `gofmt -w` before submitting. Add the smallest
regression that fails before a fix and passes afterward, retaining nearby tests.
Race tests are useful for state/concurrency changes. Report skipped or unavailable
checks accurately; skipped GitHub jobs are not passing tests.

The shared rule harness is
[internal/rules/builtin/txtar_test.go](internal/rules/builtin/txtar_test.go).
Examples such as
[ext4_journal_deprecated.txtar](internal/rules/builtin/local/ext4/testdata/ext4_journal_deprecated.txtar)
show `input`, `kernel` and `issues` sections. `input/...` can provide mock metadata;
`cache`/`options.txt` supply the harness's documented per-test observations and
rule selection. Match exact severities and findings, including unknown cases.
Keep test inputs synthetic and fixtures small enough to review.

## Test a local configuration safely

Build a local binary and use an explicitly selected harmless file:

```sh
go build -o ./vifstab ./cmd/vifstab
printf '# synthetic configuration\n' > example.fstab
./vifstab verify --static --file ./example.fstab
```

This uses the verification-first CLI. Static verification does not probe host metadata or write the selected file. Use documentation matching the checkout or installed version.

Editing, in-place formatting, RCS and experimental UEFI correlation have limits. See [save safety](docs/SAVE-SAFETY.md) and [diagram sources](docs/DIAGRAM-SOURCES.md). Never use an important configuration or real `/etc/fstab` for a demonstration.

## Report a case or submit a PR

Use the bug or TXTAR submission template for non-sensitive cases. First follow
[capture privacy review](docs/CAPTURE-PRIVACY.md). A support archive is not
automatically safe to publish. Use [SECURITY.md](SECURITY.md) for vulnerabilities
or accidentally exposed credentials; do not attach them to a public issue.

Link at least one existing issue to every PR, including documentation/design
PRs. Each issue has **one** designated resolving PR, and each implementation
PR may resolve **at most one** issue. Use exactly one `Fixes #N`,
`Closes #N`, or `Resolves #N` keyword when the issue's complete acceptance
criteria are satisfied; use `Refs #N` for dependencies, partial work,
related fixes, and design-only PRs. Do not add competing closing keywords
for the same issue in stacked or overlapping PR descriptions. When the work
spans several PRs, designate one final closer and leave the issue open until
that PR is integrated and verified. Do not mark a speculative issue complete.

Explain the behavioral change, and record focused/full test results and
limitations. Keep unrelated refactors separate. Update README
and manual together when changing the CLI, and regenerate generated files from
their authoritative definitions. Work through a draft PR while scope/tests are
unfinished, then request review. Merge, visibility, release and announcement
decisions remain with the owner.

Fork PR checks use a read-only token and no retained checkout credentials. A
maintainer must review changes before running privileged manual automation or
publishing from that revision. See [workflow security](docs/WORKFLOW-SECURITY.md).

## Development and release checks


```sh
go test ./...
go vet ./...
git diff --check
go test -race ./internal/editor ./internal/rules/... ./internal/cli
```

Validation fixtures live under `internal/rules/builtin/**/testdata/*.txtar`.
Compiled command scenarios are in `internal/cli/testdata/commands/*.txtar`;
these assert exact stdout, stderr and status, read-only file identity, forbidden
process probes and inherited editor credentials against temporary files.
They never use the host `/etc/fstab`.

The generated command tree comes from `internal/cli/commands.go`:

```sh
go generate ./...
```

Generation pins `go-subcommand` v0.0.32 and needs Go 1.26 (automatically selected
by Go's toolchain mechanism); generated application code has no runtime
`go-subcommand` dependency and is tested with Go 1.25. Timestamp and project
provenance are disabled for deterministic output. `--force` permits existing
handwritten CLI tests, the manual and legacy template to coexist; generation
does not run `--clean`. Do not edit generated files.


Use a patched distribution toolchain. Validate the release configuration with `goreleaser check`, verify notices with `python3 scripts/update-notices.py --check`, and run `goreleaser release --snapshot --clean` without publishing. Review checksums, contents and workflow permissions. Published releases require owner approval. Website builds: see [website build instructions](website/BUILD.md).
