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

Additionally, `vifstab` maintains a version history of the edited file using RCS (Revision Control System) backups, ensuring that you can always recover previous functional states if an error occurs.

If the validation fails (for example, if a mount point does not exist or a filesystem is unknown), `vifstab` categorizes the feedback into Suggestions, Warnings, and Errors. It requires critical Errors to be resolved before saving the modifications, thus preventing a system from becoming unbootable due to a malformed `fstab`.

## OPTIONS
**--backup-dir**="": Directory to store RCS backups. If omitted, `vifstab` defaults to a sensible location. It checks `$XDG_CACHE_HOME/vifstab`, then `$HOME/.cache/vifstab`, and finally falls back to `/tmp/.vifstab`.

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

The default `data=ordered` mode is not an equivalent durability replacement for `data=journal`; review data-integrity implications before changing the option. `vifstab` never rewrites the mount mode automatically.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`

Btrfs subvolume contents are not inferred from host VFS paths. Unobserved contents
remain unknown; explicit offline fixture observations retain their meaning.
