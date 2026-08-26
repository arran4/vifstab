# vifstab(8)

## NAME
vifstab - Edit fstab files with linting and versioning

## SYNOPSIS
vifstab [--backup-dir=<value>] [--file=<value>] [--generate-manpage] [--systemd-reload=<value>] [--verify] [--help|-h]
vifstab diagram <device>
vifstab format [--file=<value>]

## DESCRIPTION

`vifstab` is a utility designed to safely edit the `/etc/fstab` file (or any alternative `fstab` format file), providing a visudo-like safe editing workflow. It acts as a wrapper around your preferred text editor (defined by the `$VISUAL` or `$EDITOR` environment variables), providing a layer of safety by validating and linting the file contents before allowing the changes to be applied.

`vifstab` does not acquire privileges, nor does it invoke or implement sudo, su, doas, run0, setuid/SUID, runas, or other privilege-management mechanisms. It does not switch or infer user identities. All file reads, file writes, temporary-file operations, editor execution, RCS/versioning operations, backups, and final replacement operations run with the credentials of the `vifstab` process. Arranging sufficient permissions or an elevated invocation context is entirely the caller's responsibility.

Additionally, `vifstab` maintains a version history of the edited file using RCS (Revision Control System) backups, ensuring that you can always recover previous functional states if an error occurs.

If the validation fails (for example, if a mount point does not exist or a filesystem is unknown), `vifstab` categorizes the feedback into Suggestions, Warnings, and Errors. It requires critical Errors to be resolved before saving the modifications, thus preventing a system from becoming unbootable due to a malformed `fstab`.

## OPTIONS
**--backup-dir**="": Directory to store RCS backups. If omitted, `vifstab` defaults to a sensible location. It checks `$XDG_CACHE_HOME/vifstab`, then `$HOME/.cache/vifstab`, and finally falls back to `/tmp/.vifstab`.

**--file**="": The specific file to edit. (default: "/etc/fstab")

**--generate-manpage**: Generates this man page to `vifstab.1` in the current directory.

**--systemd-reload**="warn": Behavior when editing fstab on systemd hosts (auto|warn|never). When set to `auto`, `systemctl daemon-reload` is executed after a successful save. When set to `warn`, a diagnostic message is printed to remind the user. When set to `never`, no action or warning is given.

**--verify**: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

**--help**, **-h**: Show help information.

## SUBCOMMANDS
**diagram** `<device>`: Generates an ASCII art diagram of the filesystem stack and dependencies for the specified block device.

**format** `[--file=<value>] [-i]`: Reads fstab entries and outputs them in a formatted, column-aligned manner. Defaults to formatting `/etc/fstab` unless `--file` is specified. The `-i` flag writes the formatted output back to the file in-place (cannot be used when reading from standard input).

## ENVIRONMENT
**VISUAL**, **EDITOR**: Determines the text editor to be launched by `vifstab`. The command respects standard arguments provided in these variables. `VISUAL` takes precedence over `EDITOR`. `SUDO_EDITOR` is ignored. If neither is set, it defaults to `vi`. Editor commands are executed directly rather than through a shell. Note that `vifstab` does not acquire or manage privileges; it executes the editor using the credentials of the invoking process.

**XDG_CACHE_HOME**: Used to determine the default backup directory for RCS versioning if `--backup-dir` is not explicitly set.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`
