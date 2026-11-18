# vifstab

Check your Linux `/etc/fstab` before a configuration mistake becomes a mount or
boot problem. **vifstab** is an fstab verifier, linter and visudo-inspired editor
with versioned backups.

It checks entries, mount options, device references and filesystem compatibility.
For example, it can identify an invalid mount option before you try to use it.

**Experimental:** Linux only. Validation cannot guarantee a successful boot.
Keep independent backups and review changes before writing a system configuration.

## Get started

Download a matching Linux build from [GitHub Releases](https://github.com/arran4/vifstab/releases),
or build with a patched Go toolchain (module minimum Go 1.25):

```sh
go build -o vifstab ./cmd/vifstab
./vifstab --help
```

Try read-only verification on a disposable example:

```sh
printf '# example only\n' > example.fstab
./vifstab verify --static --file ./example.fstab
```

Run `./vifstab` to verify the local fstab without editing. Use `./vifstab edit`
only when you intend to edit; vifstab does not acquire privileges. Older releases
may have different syntax: use the help and documentation for your installed version.

## What you can do

- Verify fstab with live metadata or run host-independent static checks.
- Edit explicitly with validation, metadata-preservation policy and RCS history.
- Format entries without writing by default.
- Capture and replay reviewed TXTAR support cases offline.
- Inspect partition metadata and experimental UEFI correlations.

## Documentation

- [Installation](INSTALL.md) and [user guide](docs/USER-GUIDE.md)
- [Command reference](cmd/vifstab/manpage.md)
- [Validation and mount options](docs/VALIDATION.md)
- [Save safety and recovery](docs/SAVE-SAFETY.md)
- [Support case privacy](docs/CAPTURE-PRIVACY.md)
- [Contributing](CONTRIBUTING.md) and [security reporting](SECURITY.md)

The [documentation website](https://arran4.github.io/vifstab/) is intended to
track the default branch once GitHub Pages has been enabled and deployed.

## Licence

[BSD 3-Clause](LICENSE). See [third-party notices](THIRD_PARTY_NOTICES.md).
