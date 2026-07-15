# vifstab

**⚠️ Work in Progress**

`vifstab` is a work-in-progress utility designed to emulate `visudo` for `/etc/fstab` files. It safely edits `fstab` files with built-in linting and versioning, respecting the `$EDITOR` environment variable and saving backups using RCS.

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

Run `vifstab` to edit the default `/etc/fstab` file. Note that because this edits a system file, you typically need to run it as root or via `sudo`:

```bash
sudo vifstab
```

### Editor Selection and Privilege Escalation

Because `vifstab` is meant to be run as root (or via `sudo`/`su`), it must handle editor selection securely.

1. **`$EDITOR` / `$VISUAL` Variables:** `vifstab` respects the `$EDITOR` and `$VISUAL` environment variables. If neither is set, it typically falls back to a default system editor like `vi` or `nano`.
2. **`sudo` Considerations:** When running via `sudo`, your personal `$EDITOR` variable might not be preserved in the root environment unless configured to do so. You may need to use `sudo -E` (to preserve environment variables) or configure the `Defaults env_keep += "EDITOR VISUAL"` setting in your `sudoers` file.
3. **Security:** Running an editor as root can be a security risk (e.g., shell escapes from within `vi`). Always ensure you are using a trusted editor and environment when running `vifstab`.

### Flags

- `--backup-dir value`: Directory to store RCS backups. If not provided, it defaults to `$XDG_CACHE_HOME/vifstab` or `~/.cache/vifstab` for the current user.
- `--file value`: File to edit (default: "/etc/fstab")
- `--generate-manpage`: Generate man page for vifstab to `vifstab.1`
- `--verify`: Verify the file without opening an editor

Example:

```bash
vifstab --file /path/to/custom/fstab --backup-dir /var/backups/vifstab
```

## Linting

`vifstab` includes a modular linting registry that validates your entries before saving. It categorizes feedback into three levels:

1. **Suggestions:** Suggestions can accompany a warning or stand on their own. They represent automatically determined improvements.
2. **Warnings:** Warnings indicate issues that might cause non-critical errors, such as non-existent mount options.
3. **Errors:** Errors indicate critical issues that might cause a complete failure, such as missing block devices or unknown filesystems.

Currently built-in rules check for:
- Format validity (e.g., minimum number of fields)
- Linux filesystem validity (checking against `/proc/filesystems`)

## Environment Variables

- `EDITOR` or `VISUAL`: Defines the text editor to use when opening the fstab file.
