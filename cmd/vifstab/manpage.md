# vifstab(8)

## NAME

vifstab - Verify Linux fstab configurations and explicitly edit with linting and RCS

## SYNOPSIS

vifstab

vifstab verify [--file PATH]

vifstab edit [--file PATH] [--backup-dir DIR] [--systemd-reload auto|warn|never]

vifstab format [--file PATH|-] [--in-place|-i] [--backup-dir DIR]

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

The editor is experimental. Validation cannot prevent every boot failure.
Known save limitations include concurrent edits, symlink replacement, ownership
and extended metadata loss, and RCS basename collisions (#153–#156).
Do not describe the current save workflow as production-safe.

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
There is no `/tmp` backup fallback. Histories currently use `<basename>,v`;
use separate directories for different paths with the same basename.

**--systemd-reload MODE**: Edit-only post-save mode. `warn` (default) prints a
stale-unit diagnostic for `/etc/fstab` when systemd is active; `never` disables
handling. Explicit `auto` directly executes `systemctl daemon-reload` only after
blocking validation passes and the save completes without destination, durability
or RCS errors. Overrides and post-rename failures warn instead of reloading.
A reload failure leaves the saved file installed. Reloading does not remount.

**format**: Print aligned fstab. Only explicit **--in-place** (or **-i**) writes,
using the existing atomic-save/RCS path and sharing its known limitations.
In-place formatting requires file input and never automatically reloads systemd.

**diagram DEVICE**: Render the existing ASCII partition-table diagram, not general
mount-state reporting. Offline disk-image fixtures are supported. The legacy
block-device opening path remains a policy concern pending metadata-only
migration; do not use it to claim the normal verifier inspects device contents.

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
interface has been replaced. Static verification, TXTAR integration and edit
preflight are separately tracked work, not flags in this foundation.

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

fstab(5), mount(8), README.md

## Configuration text input safety

Fstab text inputs are classified before a data open; block-device inputs and
unavailable classification are refused. Regular files, symlinks and intentional
streams remain supported. Formatting checks borrowed stdin and retains pipelines.
The Linux pathname reader requires metadata/proc-fd interfaces and keeps caller
credentials. See `docs/TEXT-INPUTS.md` for the stable-inode boundary,
platform limits and generated CLI/TXTAR integration still pending in review branches.
