# vifstab(8)

## NAME
vifstab - Edit fstab files with linting and versioning

## SYNOPSIS
vifstab [--backup-dir=<value>] [--file=<value>] [--generate-manpage] [--verify] [--help|-h]
vifstab diagram <device>
vifstab format [--file=<value>]

## DESCRIPTION

`vifstab` is a utility designed to safely edit the `/etc/fstab` file (or any alternative `fstab` format file). It acts as a wrapper around your preferred text editor (defined by the `$VISUAL` or `$EDITOR` environment variables), providing a layer of safety by validating and linting the file contents before allowing the changes to be applied.

Additionally, `vifstab` maintains a version history of the edited file using RCS (Revision Control System) backups, ensuring that you can recover previous functional states if an error occurs.

If validation fails (for example, if a mount point does not exist or a filesystem is unknown), `vifstab` categorizes feedback into Suggestions, Warnings, and Errors and allows the user to return to the editor before saving.

`vifstab` does not acquire privileges, switch users, drop privileges, or implement setuid/runas-style behavior. File access, editor execution, and backup creation all use the credentials of the process that invoked `vifstab`. Any required privilege or permission arrangement is the caller's responsibility.

## OPTIONS
**--backup-dir**="": Directory to store RCS backups. If omitted, `vifstab` defaults to a sensible location. It checks `$XDG_CACHE_HOME/vifstab`, then `$HOME/.cache/vifstab`, and finally falls back to `/tmp/.vifstab`.

**--file**="": The specific file to edit. (default: "/etc/fstab")

**--generate-manpage**: Generates this man page to `vifstab.1` in the current directory.

**--verify**: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

**--help**, **-h**: Show help information.

## SUBCOMMANDS
**diagram** `<device>`: Generates an ASCII art diagram of the filesystem stack and dependencies for the specified block device.

**format** `[--file=<value>] [-i]`: Reads fstab entries and outputs them in a formatted, column-aligned manner. Defaults to formatting `/etc/fstab` unless `--file` is specified. The `-i` flag writes the formatted output back to the file in-place (cannot be used when reading from standard input).

## ENVIRONMENT
**VISUAL**, **EDITOR**: Determines the text editor to be launched by `vifstab`, with `VISUAL` taking precedence. Standard arguments are supported. If neither is set, `vifstab` defaults to `vi`. Editor commands are executed directly rather than through a shell.

Privilege-manager-specific variables such as `SUDO_EDITOR` are not interpreted by `vifstab`.

**XDG_CACHE_HOME**: Used to determine the default backup directory for RCS versioning if `--backup-dir` is not explicitly set.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`
