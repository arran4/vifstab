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
./vifstab --verify --file ./example.fstab
```

These commands match this branch's legacy CLI. **Bare invocation opens an
editor.** Verification does not save the selected fstab, but the live verifier
can read local metadata. For reproducible tests, prefer injected fixtures.
The generated verification-first CLI in [PR #182](https://github.com/arran4/vifstab/pull/182)
and capture/static/snapshot support in [PR #174](https://github.com/arran4/vifstab/pull/174)
are separate review work until merged; use the documentation for the exact
revision you run. Do not run proposed flags on an older binary by assumption.

Editing/in-place formatting and RCS remain experimental while save identity,
concurrency and metadata work is reviewed (#153–#156). The legacy diagram path
opens devices directly; do not use a real device as a contributor test. UEFI
correlation is experimental, and unavailable firmware data is not a boot verdict.
Never use an important configuration or real `/etc/fstab` for a demonstration.

## Report a case or submit a PR

Use the bug or TXTAR submission template for non-sensitive cases. First follow
[capture privacy review](docs/CAPTURE-PRIVACY.md). A support archive is not
automatically safe to publish. Use [SECURITY.md](SECURITY.md) for vulnerabilities
or accidentally exposed credentials; do not attach them to a public issue.

Link the relevant issue, explain the behavioral change, and record focused/full
test results and limitations. Keep unrelated refactors separate. Update README
and manual together when changing the CLI, and regenerate generated files from
their authoritative definitions. Work through a draft PR while scope/tests are
unfinished, then request review. Merge, visibility, release and announcement
decisions remain with the owner.

Fork PR checks use a read-only token and no retained checkout credentials. A
maintainer must review changes before running privileged manual automation or
publishing from that revision. See [workflow security](docs/WORKFLOW-SECURITY.md).
