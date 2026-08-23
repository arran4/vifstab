# vifstab

**⚠️ Work in Progress**

`vifstab` is a work-in-progress utility designed to emulate the safe-editing workflow of `visudo` for `/etc/fstab` files. It edits `fstab` files with built-in linting and versioning, respecting the `$VISUAL` and `$EDITOR` environment variables and saving backups using RCS.

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

### System Installation

After building from source or installing via Go, the binary can be installed to a system-wide directory such as `/usr/local/sbin` or `/usr/sbin`. Run the installation command with whatever administrative mechanism is appropriate for your system:

```bash
install -o root -g root -m 0755 vifstab /usr/local/sbin/vifstab
```

## Usage

Run `vifstab` to edit the default `/etc/fstab` file:

```bash
vifstab
```

`vifstab` does not acquire privileges, switch users, drop privileges, or implement setuid/runas-style behavior. All reads, writes, editor execution, and backup operations run with the credentials of the process that invoked `vifstab`. If the selected file is not accessible with those credentials, arrange the required permissions or invocation context outside `vifstab`.

### Editor Selection

1. **`$VISUAL` / `$EDITOR`:** `vifstab` checks `$VISUAL` first and then `$EDITOR`. If neither is set, it falls back to `vi`.
2. **Arguments:** Standard editor arguments are supported. The configured command is parsed into an executable and argument vector and executed directly rather than through a shell.
3. **Caller-owned security boundary:** The editor runs with the same process identity as `vifstab`. `vifstab` does not inspect privilege-manager-specific environment variables such as `SUDO_EDITOR` and does not attempt to determine or change the invoking user.

### Flags

- `--backup-dir value`: Directory to store RCS backups. If not provided, it defaults to `$XDG_CACHE_HOME/vifstab` or `~/.cache/vifstab` for the current user.
- `--file value`: File to edit (default: "/etc/fstab")
- `--generate-manpage`: Generate man page for vifstab to `vifstab.1`
- `--verify`: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

Example:

```bash
vifstab --file /path/to/custom/fstab --backup-dir /var/backups/vifstab
```

### Subcommands

`vifstab` provides several subcommands for specific tasks:

- `diagram <device>`: Generates an ASCII art diagram of the filesystem stack and dependencies for the given block device.
- `format [options]`: Formats the `fstab` entries into a column-aligned format.

Examples:

Generate a diagram for a block device:

```bash
vifstab diagram /dev/sda1
```

Format an existing `fstab` file:

```bash
vifstab format --file /etc/fstab
```

Format an existing `fstab` file in-place:

```bash
vifstab format -i --file /etc/fstab
```

Format `fstab` entries from standard input:

```bash
cat /etc/fstab | vifstab format
```

## Linting

`vifstab` includes a modular linting registry that validates your entries before saving. It categorizes feedback into three levels:

1. **Suggestions:** Suggestions can accompany a warning or stand on their own. They represent automatically determined improvements.
2. **Warnings:** Warnings indicate issues that might cause non-critical errors, such as non-existent mount options.
3. **Security:** Security issues are similar to warnings but track their own severity score for security-related issues.
4. **Errors:** Errors indicate critical issues that might cause a complete failure, such as missing block devices or unknown filesystems.

Currently built-in rules check for:
- Format validity (e.g., minimum number of fields)
- Linux filesystem validity (checking against `/proc/filesystems`)

## Environment Variables

- `VISUAL` or `EDITOR`: Defines the text editor to use when opening the fstab file. `VISUAL` takes precedence.
