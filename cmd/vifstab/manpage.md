# vifstab(8)

## NAME
vifstab - Edit fstab files with linting and versioning

## SYNOPSIS
vifstab [--backup-dir=<value>] [--file=<value>] [--generate-manpage] [--verify] [--help|-h]

## DESCRIPTION

`vifstab` is a utility designed to safely edit the `/etc/fstab` file (or any alternative `fstab` format file). It acts as a wrapper around your preferred text editor (defined by the `$VISUAL` or `$EDITOR` environment variables), providing a layer of safety by validating and linting the file contents before allowing the changes to be applied.

Additionally, `vifstab` maintains a version history of the edited file using RCS (Revision Control System) backups, ensuring that you can always recover previous functional states if an error occurs.

If the validation fails (for example, if a mount point does not exist or a filesystem is unknown), `vifstab` categorizes the feedback into Suggestions, Warnings, and Errors. It requires critical Errors to be resolved before saving the modifications, thus preventing a system from becoming unbootable due to a malformed `fstab`.

## OPTIONS
**--backup-dir**="": Directory to store RCS backups. If omitted, `vifstab` defaults to a sensible location. It uses `os.UserCacheDir()` (which checks `$XDG_CACHE_HOME` and falls back to `$HOME/.cache/vifstab`), and finally falls back to a secure unpredictable temporary directory if no home directory is found.

**--file**="": The specific file to edit. (default: "/etc/fstab")

**--generate-manpage**: Generates this man page to `vifstab.1` in the current directory.

**--verify**: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

**--help**, **-h**: Show help information.

## ENVIRONMENT
**VISUAL**, **EDITOR**: Determines the text editor to be launched by `vifstab`. The command respects standard arguments provided in these variables. If neither is set, it may default to common editors like `vi`.

**XDG_CACHE_HOME**: Used to determine the default backup directory for RCS versioning if `--backup-dir` is not explicitly set.

## SEE ALSO
`fstab(5)`, `mount(8)`, `rcs(1)`
