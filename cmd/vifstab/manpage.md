# vifstab(8)

## NAME
vifstab - Edit fstab files with linting and versioning

## SYNOPSIS
vifstab [--backup-dir=<value>] [--file=<value>] [--generate-manpage] [--systemd-reload=<value>] [--verify] [--help|-h]
vifstab --verify --txtar <snapshot.txtar|->
vifstab capture [--file=<value>] [--output=<snapshot.txtar|->] [--unredacted]
vifstab replay [--input=<snapshot.txtar|->]
vifstab diagram <device>
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

**--txtar**="": With `--verify`, validate the given TXTAR snapshot offline. `-` reads standard input. Requires `--verify` and cannot be combined with `--generate-manpage`.

**--help**, **-h**: Show help information.

## SUBCOMMANDS
**capture** `[--file=<value>] [--output=<snapshot.txtar|->] [--unredacted]`: Capture the fstab (default `/etc/fstab`), kernel version, filesystem support, device inventory, mount locations and observations used by production validation. Output defaults to stdout. New output files use `0600`; existing files and symlinks are rejected. Diagnostics go to stderr. Default redaction pseudonymizes identifiers, aliases, account IDs, private paths, network locations and option values consistently. `--unredacted` retains sensitive information, including possible credentials, and prints a warning.

**replay** `[--input=<snapshot.txtar|->]`: Validate a snapshot with the production rules and exit-status policy. Input defaults to stdin. Equivalent to `--verify --txtar`. Replay never consults host metadata to fill gaps or creates temporary files. Missing observations remain unknown; explicit absence and permission denial remain distinct. Editing, backups, systemd reloads, diagrams, UEFI analysis and filesystem-content inspection are unsupported replay operations.

**diagram** `<device>`: Generates an ASCII art diagram of the filesystem stack and dependencies for the specified block device.


**format** `[--file=<value>] [-i]`: Reads fstab entries and outputs them in a formatted, column-aligned manner. Defaults to formatting `/etc/fstab` unless `--file` is specified. The `-i` flag writes the formatted output back to the file in-place (cannot be used when reading from standard input).

**uefi** `<device> <mountpoint>`: Correlates an fstab ESP entry with firmware boot entries and the currently booted EFI partition. Note this subcommand depends on access to `/sys/firmware/efi` to determine EFI boot variables.

## SUPPORT SNAPSHOTS

Generate a support attachment with `vifstab capture --output case.txtar`, inspect the entire file, then use `vifstab replay --input case.txtar`. To capture an alternative fstab, add `--file ./fstab`. Pipelines use `vifstab capture --output - | vifstab replay --input -`. The caller controls permissions and overwriting when redirecting stdout.

Even redacted captures require review before public sharing. Public role mountpoints (`/`, `/boot`, `/efi`, `/boot/efi`), filesystem/partition types, option names and topology remain visible. Comments and kernel build identities are removed. Redaction of option values and malformed fields can affect validation findings. Use `--unredacted` only for private reproduction when that detail is needed. Capture never opens credential-file contents or exports account databases, reads block-device contents, mounts devices, or escalates privileges.

The versioned TXTAR contains `manifest.json` (version `1.0`, type `system`, redaction flag), `input/etc/fstab`, optional `kernel`, `provider/inventory.json` and `provider/observations.json`. Legacy `input`, `input/proc/filesystems` and line-based `cache` sections are supported. Input limits are 10 MiB total, 2 MiB per section, 100 sections and 1024 bytes per name. Unsafe paths, malformed data, ambiguous sections and unsupported versions are rejected. Unavailable metadata remains unknown rather than becoming a validation failure.

For a repository regression, copy a reviewed capture to a rule's `testdata`, append `expected/validationerrors.txt`, and optionally select `rule=<name>` (or `all`) and `format=plain` in `options.txt`. The test harness shares production snapshot handling and retains its expected-output controls.

## ENVIRONMENT
**VISUAL**, **EDITOR**: Determines the text editor to be launched by `vifstab`. The command respects standard arguments provided in these variables. `VISUAL` takes precedence over `EDITOR`. `SUDO_EDITOR` is ignored. If neither is set, it defaults to `vi`. Editor commands are executed directly rather than through a shell. Note that `vifstab` does not acquire or manage privileges; it executes the editor using the credentials of the invoking process.

**XDG_CACHE_HOME**: Used to determine the default backup directory for RCS versioning if `--backup-dir` is not explicitly set.

## EXT4 JOURNAL MODE COMPATIBILITY

Linux 7.3 deprecates the EXT4 `data=journal` mount option. When the running kernel version is 7.3 or newer, `vifstab` reports a non-blocking informational deprecation notice for that option. Before 7.3 it remains valid. Upstream plans to remove it in January 2028, but the specific removal kernel has not been confirmed; `vifstab` does not preemptively report a mount failure on a guessed release. An unknown kernel version does not trigger a deprecation notice.

The default `data=ordered` mode is not an equivalent durability replacement for `data=journal`; review data-integrity implications before changing the option. `vifstab` never rewrites the mount mode automatically.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`
