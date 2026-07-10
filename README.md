# vifstab

**⚠️ Work in Progress**

`vifstab` is a work in progress utility designed to emulate `visudo` for `/etc/fstab` files. It safely edits `fstab` files with built-in linting and versioning, respecting the `$EDITOR` environment variable and saving backups using RCS.

## Features

- **Safe editing:** Edits `/etc/fstab` with validation to prevent critical boot errors.
- **Linting:** Built-in modular rules for format and filesystem checks.
- **Versioning:** Automatically backs up changes using RCS.
- **Manpage generation:** Can generate its own man page.

## Installation

### Using Go

You can install `vifstab` using the `go install` command:

```bash
go install github.com/arran4/vifstab/cmd/vifstab@latest
```

### Building from Source

```bash
git clone https://github.com/arran4/vifstab.git
cd vifstab
go build -o vifstab ./cmd/vifstab
```

## Usage

Run `vifstab` to edit the default `/etc/fstab` file:

```bash
vifstab
```

### Flags

- `--backup-dir value`: Directory to store RCS backups (default: "/root")
- `--file value`: File to edit (default: "/etc/fstab")
- `--generate-manpage`: Generate man page for vifstab to `vifstab.1`

Example:

```bash
vifstab --file /path/to/custom/fstab --backup-dir /var/backups/vifstab
```

## Linting

`vifstab` includes a modular linting registry that validates your entries before saving. It categorizes feedback into three levels:

1. **Suggestions:** Suggestions can be with a warning or on their own. They are automatically determined improvements.
2. **Warnings:** Warnings are anything that might cause non-critical errors, such as non-critical mount point options that don't exist.
3. **Errors:** Errors are anything that might cause a complete failure, such as missing block devices, unknown filesystems, etc.

Currently built-in rules check for:
- Format validity (e.g. minimum number of fields)
- Linux filesystem validity (checking against `/proc/filesystems`)

## Environment Variables

- `EDITOR` or `VISUAL`: Defines the text editor to use when opening the fstab file.
