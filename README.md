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
- `--txtar value`: With `--verify`, validate a TXTAR snapshot instead of the live system (`-` reads standard input).
- `--verify`: Verify the file without opening an editor. Exits with a non-zero status if validation fails.

Example:

```bash
vifstab --file /path/to/custom/fstab --backup-dir /var/backups/vifstab
```

### Subcommands

`vifstab` provides several subcommands for specific tasks:

- `capture [--file /etc/fstab] [--output system.txtar] [--unredacted]`: Records the validation environment. Output defaults to standard output; files are created exclusively with mode `0600`.
- `replay [--input system.txtar]`: Validates a snapshot offline; input defaults to standard input. Equivalent to `--verify --txtar system.txtar`.
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

## Capturing a support case

```bash
vifstab capture --output system.txtar
vifstab capture --file ./fstab --output case.txtar
vifstab replay --input case.txtar
vifstab --verify --txtar case.txtar
vifstab capture --output - | vifstab replay --input -
```

Capture records the fstab, kernel version, filesystem support, block inventory,
current mount locations and the file/device/UID/GID observations requested by the
production validator. It uses kernel and filesystem metadata through the normal
provider. It never reads block-device contents, mounts a device, reads credential
files, exports account databases, or elevates privileges. An unreadable fstab is
an error; inaccessible system metadata is retained as unavailable or unknown.

Default redaction uses consistent pseudonyms for device paths, UUIDs, labels,
aliases, account IDs, personal mount paths, network sources and option values.
Public role paths (`/`, `/boot`, `/efi`, `/boot/efi`) remain visible for validation.
Kernel build identities and diagnostic details are removed. Comments are replaced.
**Review the entire archive before attaching it to a public issue.** Redaction is
not an anonymity guarantee: filesystem types, partition types, public role paths,
option names and topology remain visible. Unknown option values and malformed
fields can change meaning when sanitized, so redacted findings may differ from
live verification. Unredacted snapshots preserve more detail for exact reproduction:

```bash
vifstab capture --unredacted --output private-case.txtar
```

This prints a warning to stderr and can include credentials. Keep that file private.
Existing output paths, including symlinks, are rejected; choose a new filename.
When using stdout, the caller controls destination permissions and overwrite behavior.
Diagnostics go to stderr, keeping stdout suitable for pipelines.

Replay runs the same parser, rule registry, validation pipeline and blocking-status
policy as live verification. It reads only the archive, makes no temporary files,
and never fills missing observations from the replay host. An unrecorded check is
unknown, distinct from an explicit negative or permission denial. Full validation
is the supported replay operation; editing, backups, systemd actions, diagrams,
UEFI analysis and filesystem-content inspection are not supported replay operations.

Archives use `golang.org/x/tools/txtar` with `input/etc/fstab`, optional `kernel`,
`manifest.json` (`version: "1.0"`, `type: "system"`, `redacted: true/false`),
`provider/inventory.json` and `provider/observations.json`. Observations keep file,
block, user, group and resolution queries separate, including unknown states and
provider errors. Serialization is deterministic. Input is limited to 10 MiB,
100 sections, 2 MiB per section and 1024 bytes per archive name. Unsafe paths,
ambiguous sections, malformed structured data and unsupported versions are rejected.
Legacy fixtures using `input`, `input/proc/filesystems`, `kernel` and line-based
`cache` entries remain supported without inventing host defaults.

To make a regression fixture, copy a reviewed capture into the relevant rule's
`testdata` directory, append `expected/validationerrors.txt`, and optionally add
`options.txt` with `rule=<name>` and `format=plain` (or the existing harness's
`level_prefix`/`contains` formats). Choose `rule=all` to exercise every production
rule. The fixture harness shares the CLI snapshot parser and provider adapter;
expected output and rule-selection controls remain test-only.

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
