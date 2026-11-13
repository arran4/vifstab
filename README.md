# vifstab

**⚠️ Work in Progress**

`vifstab` is a work-in-progress utility providing a visudo-like safe editing workflow for `/etc/fstab` files. It safely edits `fstab` files with built-in linting and versioning, respects the `$VISUAL` or `$EDITOR` environment variables, and saves backups using RCS.

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

After building from source or installing via Go, it is highly recommended to install the binary to a system-wide directory such as `/usr/local/sbin` or `/usr/sbin` so it is accessible to administrators:

```bash
sudo cp vifstab /usr/local/sbin/
sudo chown root:root /usr/local/sbin/vifstab
sudo chmod 755 /usr/local/sbin/vifstab
```

## Usage

Run `vifstab` to edit the default `/etc/fstab` file:

```bash
vifstab
```
*Note: Modifying system files like `/etc/fstab` requires elevated privileges. It is the caller's responsibility to arrange sufficient permissions (e.g., using `sudo`, `doas`, or `su`) when invoking `vifstab`.*

### Security Model and Privileges

`vifstab` is designed to validate and version files, not to manage process privileges. It provides a visudo-like safe editing workflow, but unlike `visudo`, `vifstab` does not acquire or manage privileges itself; it relies entirely on the invocation context.

*   `vifstab` does **not** acquire or manage privileges.
*   `vifstab` does **not** switch users or identities.
*   `vifstab` does **not** implement sudo/su/setuid/runas-specific behavior.
*   All filesystem access, editor execution, backups, and final writes occur using the process credentials supplied by the caller.
*   Users are responsible for arranging whatever permissions or invocation context they require.
*   Editor commands are executed directly rather than through a shell.
*   Normal editor arguments are supported.

### Setting the Editor

`vifstab` respects the following environment variables (in order):
1. `VISUAL`
2. `EDITOR`

Note: `SUDO_EDITOR` is intentionally ignored because `vifstab` does not infer or manage sudo-specific environments.

### Flags

- `--backup-dir value`: Directory to store RCS backups. If not provided, it defaults to `$XDG_CACHE_HOME/vifstab` or `~/.cache/vifstab` for the current user.
- `--file value`: File to edit (default: "/etc/fstab")
- `--generate-manpage`: Generate man page for vifstab to `vifstab.1`
- `--systemd-reload value`: Post-save systemd handling for `/etc/fstab` (`auto|warn|never`) (default: `warn`). `auto` runs `systemctl daemon-reload` only after validation and a committed save; an explicitly saved validation failure is warned instead. `warn` prints a diagnostic; `never` does nothing. In-place formatting of `/etc/fstab` warns conservatively and never reloads automatically.
- `--verify`: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

Example:

```bash
vifstab --file /path/to/custom/fstab --backup-dir /var/backups/vifstab
```

### Subcommands

`vifstab` provides several subcommands for specific tasks:

- `diagram <device>`: Generates an ASCII art diagram of the filesystem stack and dependencies for the given block device.
- `format [options]`: Formats the `fstab` entries into a column-aligned format.
- `uefi <device> <mountpoint>`: Correlates an fstab ESP entry with firmware boot entries and the currently booted EFI partition.

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

### EXT4 `data=journal` compatibility

`vifstab` checks the running Linux kernel version when linting EXT4's `data=journal` mount option:

- **Before 7.3:** `data=journal` remains valid.
- **7.3 and later:** a non-blocking informational notice explains that it is deprecated and upstream plans removal in January 2028.
- **Removal:** the precise kernel version is not yet confirmed, so `vifstab` does **not** reject the option solely on a guessed future version or calendar date. Revisit the rule when the removal lands.

The informational notice does not rewrite `fstab`. Switching to the default `data=ordered` changes crash-consistency guarantees, so the administrator must decide whether it is an appropriate alternative. If the kernel version cannot be determined, this rule does not assume deprecation is active.

Currently built-in rules check for:
- Format validity (e.g., minimum number of fields)
- Linux filesystem validity (checking against `/proc/filesystems`)

## Environment Variables

- `VISUAL` or `EDITOR`: Defines the text editor to use when opening the fstab file. `VISUAL` takes precedence over `EDITOR`. `SUDO_EDITOR` is not interpreted. Normal editor arguments are supported. Editor commands are executed directly rather than through a shell.

## License

`vifstab` is licensed under the [BSD 3-Clause License](LICENSE).

## Configuration text input safety

Fstab text inputs are classified before a data open; block-device inputs and
unavailable classification are refused. Regular files, symlinks and intentional
streams remain supported. Formatting checks borrowed stdin and retains pipelines.
The Linux pathname reader requires metadata/proc-fd interfaces and keeps caller
credentials. See [text input sources and limits](docs/TEXT-INPUTS.md) for the stable-inode boundary,
platform limits and generated CLI/TXTAR integration still pending in review branches.
