# vifstab(8)

## NAME

vifstab - Verify Linux fstab configurations and explicitly edit with linting and RCS

## SYNOPSIS

vifstab

vifstab verify [--file PATH|-] [--static] [--capture FILE|-] [--unredacted]

vifstab verify --snapshot FILE|-

vifstab edit [--file PATH] [--backup-dir DIR] [--systemd-reload auto|warn|never] [--metadata-policy strict|supported|best-effort]

vifstab format [--file PATH|-] [--in-place|-i] [--backup-dir DIR] [--metadata-policy strict|supported|best-effort]

vifstab diagram DEVICE

vifstab uefi DEVICE MOUNTPOINT

vifstab manpage [--output vifstab.1]

vifstab version

vifstab help [COMMAND]

## DESCRIPTION

Bare invocation always verifies `/etc/fstab` read-only, regardless of credentials.
`verify` uses the same action. Verification creates no backups, opens no editor,
performs no systemd reload and needs no cache directory. It uses available live
metadata, even for an alternative file. Unknown observations are not definite
absence. Verification does not establish bootability or whether an entry is
currently mounted. Linux is the immediate platform; general Linux 7-series
compatibility is a development goal and other fstab dialects are future work.

This repository is in a verification and expansion stage, not a production-proven
safety claim. Validation cannot prevent every boot failure. Saves resolve
symlinks to their regular target without replacing the symlink, detect stale
concurrent edits (preserving a `0600` recovery copy), isolate RCS histories by
resolved target path, and preserve ownership, mode, special bits, and xattrs
according to `--metadata-policy` (`strict` default, `supported`, `best-effort`).

## OPTIONS AND COMMANDS

**--file PATH**: Input fstab, default `/etc/fstab`. For format, explicit file
selection takes precedence over piped stdin; `-` explicitly selects stdin.
Without an explicit format file, non-terminal stdin is used if available,
otherwise the default file is read. Input-probe failures report an error.

**edit**: Explicitly open a temporary copy in the configured editor, validate it,
then save or prompt to re-edit, override blocking findings or abort.
An editor or save failure never silently falls back to verification.

**--backup-dir DIR**: RCS directory for edit and in-place format. Default:
`$XDG_CACHE_HOME/vifstab`, otherwise the caller's user cache directory (normally
`$HOME/.cache/vifstab`). If unavailable, an explicit directory is required.
There is no `/tmp` backup fallback. Histories are isolated by resolved target path.

**--metadata-policy MODE**: Metadata preservation policy for `edit` and
`format --in-place` (`strict` default, `supported`, `best-effort`). Rejected on
read-only `format` and unavailable on read-only commands.

**--systemd-reload MODE**: Edit-only post-save mode. `warn` (default) prints a
stale-unit diagnostic for `/etc/fstab` when systemd is active; `never` disables
handling. Explicit `auto` directly executes `systemctl daemon-reload` only after
blocking validation passes and the save completes without destination, durability
or RCS errors. Overrides and post-rename failures warn instead of reloading.
A reload failure leaves the saved file installed. Reloading does not remount.

**format**: Print aligned fstab. Only explicit **--in-place** (or **-i**) writes,
using the atomic-save/RCS path (`--metadata-policy` defaults to `strict`).
In-place formatting requires file input and never automatically reloads systemd.

**diagram DEVICE**: Render an ASCII partition-table diagram from kernel metadata
interfaces without opening live block devices; disk-image inputs are guarded by
stable descriptors. Not a general mount-state reporter.

**uefi DEVICE MOUNTPOINT**: Explicit experimental live correlation through the
canonical inventory and native EFI parsing. Firmware boot entries and BootCurrent
can be correlated with PARTUUID; this is not a complete current-loader or
bootability guarantee. Unavailable firmware data and unmounted filesystem contents
remain unknown. Missing filesystem type and missing or malformed GPT type also
remain unknown; a definite non-ESP report requires a valid obtained GPT type.
Optional already-mounted EFI-tree observations are informational.
No systemd or external boot utility is required. No `verify --uefi` option exists.

**manpage --output PATH**: Explicitly write the manual, default `vifstab.1`.

**version**: Print build version, commit and date.

**help COMMAND**, **--help**, **-h**: Show command-specific help without executing
the action. Verification flags belong after `verify`; root verification flags
before another command are rejected. Flags precede positional operands. The old root editing/`--verify`
interface has been replaced. **verify --static** runs host-independent rule checks while all live filesystem,
device, path/account and kernel observations remain unchecked/unknown.
The mode is printed explicitly; unknown is not a failure and no host fallback
is used. Structural/option errors still block. Edit preflight is not implemented.

**verify --capture FILE** saves a bounded support archive from the same validation
pass, even when blocking findings or a bounded parser failure cause exit 1.
Unreadable/oversized input or archive-output failures are errors. `--file -`
explicitly borrows stdin. Capture files use private 0600 exclusive creation;
existing files and symlinks are rejected. `--capture -` writes only TXTAR to stdout
and sends findings/diagnostics to stderr; shell redirection controls permissions.
Static capture is supported without live observations.

Capture uses the canonical metadata provider; it does not inspect block contents,
mount devices, export account databases, read credentials or elevate privileges.
Default redaction pseudonymises identities, paths, network sources and sensitive
options; topology and public role paths remain visible. **Manually review every
capture before sharing.** Redaction can alter validation semantics/findings.
**--unredacted** requires capture, prints an explicit warning and may retain
credentials. Keep such artifacts private.

**verify --snapshot FILE** (or `-` for stdin) runs the production validator offline.
Missing observations remain unknown and never fall back to the current host.
It does not create temporary files, backups or target writes, open an editor or
launch systemd. Snapshot mode rejects file/capture/static/unredacted combinations.
UEFI, diagram, editing and content inspection are unsupported replay operations.

TXTAR schema 1.0 includes manifest.json, input/etc/fstab, optional kernel and
provider inventory/observations JSON. Negative/unknown/permission states remain
distinct. Serialization is deterministic; limits are 10 MiB total, 100 sections,
2 MiB per section and 1024-byte names. Unsafe paths, ambiguous/duplicate sections,
malformed/duplicate JSON, unsupported versions and delimiters are rejected.
Legacy fixture forms remain supported. To add a regression, copy a reviewed
capture to testdata, add expected/validationerrors.txt and optional options.txt
with rule=all and format=level_prefix. See docs/USER-GUIDE.md and CONTRIBUTING.md for executable examples.

## PROCESS AND SAVE BOUNDARIES

All operations inherit the caller's credentials. vifstab never acquires privileges,
changes UID/GID or launches an elevation helper. Its only subprocess categories
are the user-configured editor from explicit edit and an eligible post-save
systemctl reload with explicit auto mode. No shell is used to launch the editor.

The save path writes a same-directory temporary file, applies permission bits,
syncs the file, closes it, atomically renames it and syncs the directory.
A later durability, backup or systemd error can return failure after the new fstab
is installed. Inspect the diagnostic before retrying. Current RCS saves do not
provide an unconditional guarantee of recovery of an earlier functional system.

## FINDINGS AND STATUS

`warning`, `security` and `error` block verification; `info` does not.
Suggestions may occur in diagnostic text. Rule families include structure,
filesystem support, sources/targets, device identity/partition roles and
filesystem-specific options. Accepted extension options may remain unverified.

Exit 0 means completion, including explicit abort. Exit 1 indicates blocking
findings, usage errors or operational failure. A post-rename error does not imply
that the destination still contains its old data.

EXT4 data=journal emits non-blocking deprecation information for a known kernel
of 7.3 or newer. An unknown kernel version or guessed removal date is not proof
that the option is unsupported. Review integrity differences before selecting
data=ordered; vifstab does not rewrite the mode automatically.

## ENVIRONMENT

**VISUAL**, **EDITOR**: Direct editor command with quoted arguments, in that
priority order, default `vi`. Arguments `--` and the temporary path are appended.
`SUDO_EDITOR` is ignored. Standard streams and caller credentials are inherited.

**XDG_CACHE_HOME**, **HOME**: Caller cache lookup for writing workflows.

## SEE ALSO

fstab(5), mount(8), README.md, docs/USER-GUIDE.md

## Configuration text input safety

Fstab text inputs are classified before a data open; block-device inputs and
unavailable classification are refused. Regular files, symlinks and intentional
streams remain supported. Formatting checks borrowed stdin and retains pipelines.
The Linux pathname reader requires metadata/proc-fd interfaces and keeps caller
credentials. See `docs/VALIDATION.md` for the stable-inode boundary,
platform limits. Generated verification, static input, capture and replay share this guard.

## USER AND GROUP OBSERVATIONS

Numeric UID/GID mount options must be between `0` and `4294967294`. Explicitly supported filesystem enum alternatives remain valid. A structurally valid matching local passwd/group record establishes presence; missing, unreadable, malformed or unmatched local data leaves account availability unknown because NSS may use other sources. Unknown and permission-denied observations produce informational diagnostics. Explicit provider/cache observations may still establish absence. No additional account-discovery command or NSS lookup is performed.
