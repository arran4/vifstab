# vifstab

`vifstab` verifies Linux fstab configurations and reports validity, safety,
filesystem compatibility and configuration mistakes. It also provides an explicit
visudo-inspired editing workflow, pure-Go RCS versioning and optional post-save
systemd handling. This repository is in a **verification and expansion** stage,
not a production-proven safety claim: validation cannot prevent every boot
failure.

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
against this machine unless `--static` or `--snapshot` is explicitly selected.
Missing observations are not proof of absence.

Linux is the immediate platform. General Linux 7-series compatibility is a
development goal, not a guarantee. BSD and other fstab dialects are future work.

## Commands

```text
vifstab
vifstab verify [--file PATH|-] [--static] [--capture FILE|-] [--unredacted]
vifstab verify --snapshot FILE|-
vifstab edit [--file PATH] [--backup-dir DIR] [--systemd-reload MODE] [--metadata-policy MODE]
vifstab format [--file PATH|-] [--in-place|-i] [--backup-dir DIR] [--metadata-policy MODE]
vifstab diagram DEVICE
vifstab uefi DEVICE MOUNTPOINT
vifstab manpage [--output vifstab.1]
vifstab version
vifstab help [COMMAND]
```

Each command supports `--help` and `-h`, and `vifstab help COMMAND` shows its help
without performing the action. Flags precede positional operands. `--file`
defaults to `/etc/fstab`. Verification options belong after `verify`; root flags before another command are rejected.
The earlier root `--verify`/editing interface has been
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
`/tmp` backup fallback. RCS histories are isolated by resolved target path.

Saves resolve symlinks to their regular target without replacing the symlink,
detect concurrent external edits (refusing overwrite and preserving a `0600`
recovery copy), and preserve ownership, permissions, special bits, and xattrs
under `--metadata-policy`: `strict` (default, fails before rename if required
metadata cannot be observed, applied, or reverified), `supported` (allows
`ENOTSUP` xattrs while still requiring ownership and mode), and `best-effort`
(applies supported metadata and emits warnings for unpreserved attributes).
A post-rename error may leave the new content installed even though the command
returns failure; inspect the diagnostic before retrying.

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

`--in-place` (also `-i`) uses the atomic save/RCS path and requires file input.
`--metadata-policy` (`strict|supported|best-effort`, default `strict`) applies to
`--in-place` saves and is rejected on read-only `format`. In-place formatting
never automatically reloads systemd and conservatively warns for `/etc/fstab`.

### Target Ownership and Permissions Verification

During verification and editing, `vifstab` inspects the selected effective target
(`--file`, following symlinks via `stat`) once per file—including empty fstab
files—for ownership, mode bits, and Linux POSIX access ACLs (`system.posix_acl_access`):
- A system target (`/etc/fstab`) triggers a warning if it is world-writable,
  owned by an unexpected non-root user, group-writable by a non-root group, or
  has ACL write access granted to non-root users.
- Alternate user-specified fstab paths are warned if world-writable or if write
  ACLs expose write access to non-owners.
- Secure configurations (`0644` or `0600` root:root) produce no warnings.
- Unavailable, unsupported, or unreadable metadata remains unknown rather than
  emitting false positives, and `vifstab` never modifies permissions automatically.
- Permission issues use `warning` severity, which is blocking: verification
  exits non-zero, and `vifstab edit` prompts to re-edit, override, or abort.

### Diagrams and experimental UEFI correlation

`diagram DEVICE` renders an ASCII partition-table diagram from kernel metadata
interfaces without opening live block devices; disk-image file inputs are
guarded by stable descriptors. It is not a general filesystem-stack or
mount-state inspector.

`uefi DEVICE MOUNTPOINT` is an explicit experimental live report, retained while
its integration into verification is assessed. It uses the canonical device
inventory and native EFI-variable parsing, without systemd or external boot
utilities. It correlates firmware boot entries and BootCurrent with a PARTUUID;
it does not establish every current-loader configuration or bootability.
Unavailable firmware information, missing filesystem metadata and missing or
malformed GPT type metadata remain unknown. A definite non-ESP report requires
a valid obtained GPT type. Unmounted filesystem contents remain unknown. Optional observations of an already-mounted EFI tree do not gate
verification. There is no `verify --uefi` flag yet.

### Static verification

`verify --static` runs the host-independent portions of the production rules.
Live filesystem support, device inventory, path/account existence and kernel
version observations remain unchecked/unknown. The CLI prints that limitation;
unknown data is not a validation failure. Syntax errors still block verification.
It does not substitute a guessed kernel or this host's `/proc`, `/sys` or accounts.
`edit --check` is not implemented: edit preflight guarantees remain a separate
design choice, not another validator.

### Capture, privacy and offline replay

```sh
vifstab verify --file ./fstab --capture case.txtar
vifstab verify --snapshot case.txtar
vifstab verify --static --file ./fstab --capture syntax-case.txtar
vifstab verify --file - --static < ./fstab
vifstab verify --file ./fstab --capture - | vifstab verify --snapshot -
```

Capture records the observations requested by the production validator, the
relevant inventory and filesystem support, kernel version and selected fstab.
It does not read device contents, mount devices, read credential files, export
account databases or elevate privileges. Optional existing mount locations come
from the inventory. Unavailable observations remain unknown. An unreadable or
oversized input cannot be captured; ordinary blocking findings and bounded
technical validation failures still produce the support archive before returning
failure. Exit status remains meaningful even when the archive was saved.

Capture output files are created privately (`0600`) and exclusively. Existing
paths, including symlinks, are rejected. With `--capture -`, stdout contains only
the TXTAR and diagnostics/findings go to stderr. Shell redirection controls its
own output permissions and overwrite behaviour. `--file -` explicitly borrows
stdin; it is not selected just because stdin is piped.

Default sanitisation consistently pseudonymises device paths, UUID/PARTUUID/LABEL
identities, aliases, account IDs, personal paths, network sources and sensitive
option values. Kernel build identities, comments and diagnostic details are
sanitised. Public role paths (`/`, `/boot`, `/efi`, `/boot/efi`), filesystem and
partition types, option names and topology remain visible. **Review the entire
capture before public sharing.** Redaction is not an anonymity guarantee and can
change option semantics/findings. An unredacted capture preserves more detail
for exact reproduction and may contain credentials:

```sh
vifstab verify --file ./fstab --capture private-case.txtar --unredacted
```

Keep that explicitly warned capture private. `--unredacted` requires capture.

Replay uses the same parser, production rules and blocking-severity policy. It
reads the bounded archive only, creates no temporary files or backups and never
fills missing observations from the replay host. Unrecorded, explicitly absent
and permission-denied observations remain distinct. Snapshot mode cannot be
combined with `--file`, `--capture`, `--static`, edit or systemd options. Firmware
analysis, diagrams, editing and filesystem contents are unsupported replay
operations. The experimental live `uefi` report is retained separately.

The schema uses `golang.org/x/tools/txtar`: `input/etc/fstab`, optional `kernel`,
`manifest.json` (`version: "1.0"`, `type: "system"`, `redacted: true/false`),
`provider/inventory.json` and `provider/observations.json`. Files, blocks, users,
groups and resolved references have separate observations. Serialization is
deterministic. Limits: 10 MiB total, 100 sections, 2 MiB per section and 1024 bytes
per archive name. Unsafe paths, duplicate/conflicting sections, malformed data,
duplicate JSON fields, unsupported versions and delimiter injection are rejected.
Legacy `input`, `input/proc/filesystems`, `kernel` and line-based `cache` fixtures
remain supported without inventing live defaults.

For a regression, copy a manually reviewed capture into the relevant rule's
`testdata` directory, append `expected/validationerrors.txt`, and optionally add
`options.txt` with `rule=<name>` and `format=plain`, `level_prefix` or `contains`.
Use `rule=all` to exercise the complete production registry. The fixture harness
shares snapshot parsing and provider construction with the CLI; expected output
and rule selection remain test-only. Run the representative existing case:

```sh
go test ./internal/rules/builtin -run TestAllTxtarRules/testdata/captured-esp-mismatch.txtar -v
go test ./internal/txtar ./internal/cli
```

## Findings and exit status

The blocking levels are `warning`, `security` and `error`. `info` is non-blocking.
Suggestions may appear in messages; they are not a separate blocking level.
The modular rule families include structure, filesystem support, mount sources
and targets, device identity/partition roles and filesystem-specific options.
An accepted extension option is not necessarily semantically verified.

Exit 0 indicates successful completion (including an explicit abort). Exit 1
indicates blocking verification findings, usage errors or operational failures.
Verification and replay use the same blocking-severity policy. Read the diagnostics: a save may already be installed when a later
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

## Configuration text input safety

Fstab text inputs are classified before a data open; block-device inputs and
unavailable classification are refused. Regular files, symlinks and intentional
streams remain supported. Formatting checks borrowed stdin and retains pipelines.
The Linux pathname reader requires metadata/proc-fd interfaces and keeps caller
credentials. See [text input sources and limits](docs/TEXT-INPUTS.md) for the stable-inode boundary,
platform limits. Generated verification, static input, capture and replay share this guard.

### User and group observations

Numeric UID/GID mount options must be between `0` and `4294967294`; an option's explicitly supported enum values remain separate. A valid matching local passwd/group record can establish presence. Missing, unreadable, malformed or unmatched local files cannot establish global absence because NSS may use other sources. These unavailable observations produce informational diagnostics. Explicit provider/cache observations can still establish absence. See [account observation limits](docs/ACCOUNT-OBSERVATIONS.md).
