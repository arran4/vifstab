# vifstab Agent Skill

`vifstab` is a utility for safely editing `/etc/fstab` with linting and validation. As an AI agent, you should be aware of several operational specifics to use it effectively.

## Core Purpose

`vifstab` acts like `visudo` but for `fstab`. It spawns an editor (defined by `$EDITOR` or `$VISUAL`) to let users or scripts edit the fstab, lints the result, and if valid, saves it. It also automatically backs up the old fstab via RCS.

## Non-Interactive Operation

As an AI agent, you often run non-interactively. By default, `vifstab` requires an interactive terminal because it opens a visual editor (like `vi`).
To use `vifstab` programmatically, you should use an inline editor script (e.g. `sed`) by setting the `EDITOR` variable.

**Example for Agents:**
```bash
EDITOR="sed -i 's/old/new/'" vifstab
```
Wait, the editor must edit the file in place and return exit code 0.
```bash
EDITOR="sed -i 's/ext4/btrfs/'" sudo -E vifstab --file /etc/fstab
```
If linting fails, `vifstab` exits with an error and *does not* overwrite the original file. This makes it safe.

## Authentication and Sudo

Since `/etc/fstab` is owned by root, `vifstab` must typically be run as root.
Ensure you use `sudo -E vifstab` to preserve the `EDITOR` environment variable, or configure sudoers to keep it.

## Common Command Ambiguities

- `vifstab` does NOT accept arbitrary arguments. If you provide any arguments without flags, it will fail.
  Correct: `vifstab --file /tmp/fstab`
  Incorrect: `vifstab /tmp/fstab`
- Global flags must precede subcommands if applicable (though `skill` is the main subcommand framework added later, `vifstab skill` is the entrypoint for skills).

## Validation and Linting Traps

`vifstab` will block changes if it detects:
- Missing block devices
- Unknown filesystems (it checks `/proc/filesystems`)
- Malformed fstab format

If you are generating a new fstab entry in a test environment, make sure the filesystem type exists in the test system, otherwise `vifstab` will reject it as an Error.

## Dry-run / Safe mutations

To test an fstab modification without altering the real system fstab, you can use the `--file` flag pointing to a copy, and `--backup-dir` to an isolated directory.
```bash
cp /etc/fstab /tmp/fstab.test
EDITOR="sed -i '...'" vifstab --file /tmp/fstab.test --backup-dir /tmp/vifstab-backups
```

## Environment Variables

- `EDITOR` / `VISUAL`: crucial for agent automation.
- `XDG_CACHE_HOME`: determines the default backup directory if `--backup-dir` is not provided.
