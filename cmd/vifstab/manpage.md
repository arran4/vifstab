# vifstab(8)

## NAME
vifstab - Edit fstab files with linting and versioning

## SYNOPSIS
vifstab [--backup-dir=<value>] [--file=<value>] [--generate-manpage] [--systemd-reload=<value>] [--verify] [--help|-h]
vifstab diagram <device-or-image>
vifstab format [--file=<value>]
vifstab uefi <device> <mountpoint>

## DESCRIPTION

`vifstab` is a utility designed to safely edit the `/etc/fstab` file (or any alternative `fstab` format file), providing a visudo-like safe editing workflow. It acts as a wrapper around your preferred text editor (defined by the `$VISUAL` or `$EDITOR` environment variables), providing a layer of safety by validating and linting the file contents before allowing the changes to be applied.

`vifstab` does not acquire privileges, nor does it invoke or implement sudo, su, doas, run0, setuid/SUID, runas, or other privilege-management mechanisms. It does not switch or infer user identities. All file reads, file writes, temporary-file operations, editor execution, RCS/versioning operations, backups, and final replacement operations run with the credentials of the `vifstab` process. Arranging sufficient permissions or an elevated invocation context is entirely the caller's responsibility.

Additionally, `vifstab` can maintain RCS revisions of saved content. This is not a guarantee that earlier functional states are available or restorable: backup identity, concurrency and recovery limitations remain under review, along with destination symlink and metadata handling. Treat editing and in-place formatting as experimental.

Findings have info, warning, security and error levels. Info is non-blocking; warnings, security findings and errors fail verification. Editing prompts on blocking findings, and the caller can explicitly override them. Unavailable observations are not proof of absence. Validation does not guarantee successful boot or safe filesystem contents.

## OPTIONS
**--backup-dir**="": Directory to store RCS backups. If omitted, it uses `$XDG_CACHE_HOME/vifstab` or the caller's user-cache directory plus `/vifstab` (normally `$HOME/.cache/vifstab`). If that directory cannot be determined, editing requires an explicit `--backup-dir`; there is no `/tmp/.vifstab` fallback.

**--file**="": The specific file to edit. (default: "/etc/fstab")

**--generate-manpage**: Generates this man page to `vifstab.1` in the current directory.

**--systemd-reload**="warn": Post-save systemd handling for `/etc/fstab` (auto|warn|never). `auto` runs `systemctl daemon-reload` only after blocking validation succeeds and the file and backup are committed. If blocking validation is overridden, a diagnostic is printed instead of reloading. `warn` prints a diagnostic without reloading. `never` takes no systemd action and prints no systemd diagnostic. A reload failure is reported after the saved fstab is left in place. In-place formatting of `/etc/fstab` uses the conservative diagnostic and never reloads automatically.

**--verify**: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

**--help**, **-h**: Show help information.

## SUBCOMMANDS
**diagram** `<device-or-image>`: Shows the available kernel/udev partition layout for a block device without opening device contents. Missing or malformed observations remain unknown; unmarked ranges do not prove free space. Regular images retain read-only GPT/MBR table parsing through a checked stable file handle. This is not a complete filesystem-stack or filesystem-content verifier. See docs/DIAGRAM-SOURCES.md for Linux interface requirements and fixture tests.

**format** `[--file=<value>] [-i]`: Reads fstab entries and outputs them in a formatted, column-aligned manner. Defaults to formatting `/etc/fstab` unless `--file` is specified. The `-i` flag writes the formatted output back to the file in-place (cannot be used when reading from standard input).

**uefi** `<device> <mountpoint>`: Correlates an fstab ESP entry with firmware boot entries and the currently booted EFI partition. Note this subcommand depends on access to `/sys/firmware/efi` to determine EFI boot variables.

## ENVIRONMENT
**VISUAL**, **EDITOR**: Determines the text editor to be launched by `vifstab`. The command respects standard arguments provided in these variables. `VISUAL` takes precedence over `EDITOR`. `SUDO_EDITOR` is ignored. If neither is set, it defaults to `vi`. Editor commands are executed directly rather than through a shell. Note that `vifstab` does not acquire or manage privileges; it executes the editor using the credentials of the invoking process.

**XDG_CACHE_HOME**: Used to determine the default backup directory for RCS versioning if `--backup-dir` is not explicitly set.

## EXT4 JOURNAL MODE COMPATIBILITY

Linux 7.3 deprecates the EXT4 `data=journal` mount option. When the running kernel version is 7.3 or newer, `vifstab` reports a non-blocking informational deprecation notice for that option. Before 7.3 it remains valid. Upstream plans to remove it in January 2028, but the specific removal kernel has not been confirmed; `vifstab` does not preemptively report a mount failure on a guessed release. An unknown kernel version does not trigger a deprecation notice.

## TARGET OWNERSHIP AND PERMISSIONS VERIFICATION

During verification and editing, `vifstab` inspects the selected effective target (`--file`, following symlinks via `stat`) once per file—including empty fstab files—for ownership, mode bits, and Linux POSIX access ACLs (`system.posix_acl_access`):
- A system target (`/etc/fstab`) triggers a warning if it is world-writable, owned by an unexpected non-root user, group-writable by a non-root group, or has ACL write access granted to non-root users.
- Alternate user-specified fstab paths are warned if world-writable or if write ACLs expose write access to non-owners.
- Secure configurations such as `0644` or `0600` root:root ownership produce no warnings.
- Unavailable, unsupported, or unreadable metadata (including ACLs) remains unknown rather than emitting false positives, and `vifstab` never modifies permissions automatically.
- Permission issues are reported at `warning` severity, which is blocking in `vifstab`: read-only verification exits non-zero, and `vifstab edit` prompts `(e)dit again, (c)ontinue and save anyway, (q)uit without saving`.


The default `data=ordered` mode is not an equivalent durability replacement for `data=journal`; review data-integrity implications before changing the option. `vifstab` never rewrites the mount mode automatically.

Mount-option validation preserves repeated values and their input order. Existing
per-option validators retain their last-value context; new cross-option checks
can inspect every occurrence. This does not by itself validate systemd extensions
or change existing filesystem conflict policies. Formatting keeps option spelling
and order. See docs/MOUNT-OPTIONS.md in the source distribution.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`

Btrfs subvolume contents are not inferred from host VFS paths. Unobserved contents
remain unknown; explicit offline fixture observations retain their meaning.

## Configuration text input safety

Fstab text inputs are classified before a data open; block-device inputs and
unavailable classification are refused. Regular files, symlinks and intentional
streams remain supported. Formatting checks borrowed stdin and retains pipelines.
The Linux pathname reader requires metadata/proc-fd interfaces and keeps caller
credentials. See `docs/TEXT-INPUTS.md` for the stable-inode boundary,
platform limits and integration notes for the unmerged review branches.
