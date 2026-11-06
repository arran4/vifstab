# vifstab

`vifstab` verifies Linux fstab configurations and reports validity, safety,
filesystem compatibility and configuration mistakes. It also provides an explicit
visudo-inspired editing workflow, pure-Go RCS versioning and optional post-save
systemd handling. This is work in progress: validation cannot prevent every boot
failure, and editor save-safety limitations remain under review.

## Build and run

The application targets Go 1.25. Use a patched Go toolchain when building binaries.

```sh
go build -o vifstab ./cmd/vifstab
./vifstab
./vifstab verify --file ./example.fstab
```

Bare invocation always verifies `/etc/fstab` read-only, including when run as root.
`verify` is an explicit route to the same action. Verification does not launch an
editor, create backups or reload systemd; it does not require a cache directory.
It uses available live metadata, so verifying a custom file still compares it
against this machine. Missing observations are not proof of absence.

Linux is the immediate platform. General Linux 7-series compatibility is a
development goal, not a guarantee. BSD and other fstab dialects are future work.

## Commands

```text
vifstab [--file PATH]
vifstab verify [--file PATH]
vifstab edit [--file PATH] [--backup-dir DIR] [--systemd-reload MODE]
vifstab format [--file PATH|-] [--in-place|-i] [--backup-dir DIR]
vifstab diagram DEVICE
vifstab uefi DEVICE MOUNTPOINT
vifstab manpage [--output vifstab.1]
vifstab version
vifstab help [COMMAND]
```

Each command supports `--help` and `-h`, and `vifstab help COMMAND` shows its help
without performing the action. Flags precede positional operands. `--file`
defaults to `/etc/fstab`. The earlier root `--verify`/editing interface has been
replaced by explicit commands; there is no established released CLI contract.

### Editing and backups

Only explicit `vifstab edit` opens an editor. It uses `VISUAL`, then `EDITOR`, then
`vi`, with normal quoted arguments. It executes the editor directly, passes `--`
and the temporary file path, and inherits the caller's credentials and standard
streams. `SUDO_EDITOR` is ignored. An editor failure or unavailable write access
is an error, not a request to fall back to verification.

After editing, blocking findings prompt for re-edit, override-save or abort.
Information does not block saving. `--backup-dir` overrides the default
`$XDG_CACHE_HOME/vifstab` or the caller's user cache directory (normally
`$HOME/.cache/vifstab`). If that directory cannot be determined, editing and
in-place formatting require an explicit backup directory; there is no shared
`/tmp` backup fallback. The current RCS history name is `<basename>,v`.

Known editor limitations tracked in #153–#156 include stale concurrent edits,
symlink replacement, ownership/extended metadata preservation and backup-name
collisions. Use a separate backup directory for distinct files sharing a basename.
The save path uses a same-directory temporary file, file sync, atomic rename and
directory sync. A post-rename error may leave the new content installed even
though the command returns failure; inspect the diagnostic before retrying.
Do not treat this experimental editor as a production-safe replacement workflow.

`--systemd-reload=warn` (default) prints a stale-unit diagnostic after saving the
real `/etc/fstab` when systemd is active. `never` disables this handling. Explicit
`auto` runs `systemctl daemon-reload` only after a completed save with blocking
validation passed and no save, durability or RCS error. An override-save or
post-rename error warns instead of automatically reloading. Reload failures leave
the saved fstab in place. Reloading does not remount filesystems.

`vifstab` never acquires privileges, changes UID/GID or internally launches an
elevation helper. The only application subprocesses are the configured editor
from explicit `edit` and the eligible, explicitly selected post-save reload.
The caller must arrange any permissions required for editing system files.

### Formatting

Formatting prints column-aligned fstab by default. An explicit `--file` takes
precedence over piped stdin; `--file -` explicitly selects stdin. Without an
explicit file, piped/file stdin is used, otherwise `/etc/fstab` is read.

```sh
vifstab format --file ./fstab
cat ./fstab | vifstab format
vifstab format --file ./fstab --in-place --backup-dir ./backups
```

`--in-place` (also `-i`) uses the existing atomic save/RCS path and requires file
input. It shares the editor's known save limitations. It never automatically
reloads systemd and conservatively warns for `/etc/fstab`.

### Diagrams and experimental UEFI correlation

`diagram DEVICE` currently renders an ASCII partition-table diagram; it is not
a general filesystem-stack or mount-state inspector. Its existing disk-image
fixtures are tested offline. The legacy block-device opening path remains a
policy concern pending migration to metadata-only device observations.

`uefi DEVICE MOUNTPOINT` is an explicit experimental live report, retained while
its integration into verification is assessed. It uses the canonical device
inventory and native EFI-variable parsing, without systemd or external boot
utilities. It correlates firmware boot entries and BootCurrent with a PARTUUID;
it does not establish every current-loader configuration or bootability.
Unavailable firmware information and unmounted filesystem contents remain
unknown. Optional observations of an already-mounted EFI tree do not gate
verification. There is no `verify --uefi` flag yet.

Static verification, TXTAR capture/replay integration and `edit --check` are
separately tracked work, not options implemented by this CLI foundation.

## Findings and exit status

The blocking levels are `warning`, `security` and `error`. `info` is non-blocking.
Suggestions may appear in messages; they are not a separate blocking level.
The modular rule families include structure, filesystem support, mount sources
and targets, device identity/partition roles and filesystem-specific options.
An accepted extension option is not necessarily semantically verified.

Exit 0 indicates successful completion (including an explicit abort). Exit 1
indicates blocking verification findings, usage errors or operational failures.
Verification and replay use the same blocking-severity policy when replay is
integrated. Read the diagnostics: a save may already be installed when a later
RCS/durability/systemd operation fails.

EXT4 `data=journal` emits non-blocking deprecation information for a known Linux
kernel of 7.3 or newer. Unknown kernel versions do not trigger it. The precise
removal kernel is not confirmed; a calendar date is not proof of unsupported
syntax. `data=ordered` has different integrity guarantees and is not silently
substituted.

## Development and regression cases

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

## License

[BSD 3-Clause](LICENSE).
