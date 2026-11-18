# Installing a reviewed vifstab build

The prepared archive targets Linux amd64 or arm64. Select the architecture for
your machine, verify its checksum against the accompanying checksums file, and
extract into a new directory. Archives contain the binary, README, this guide,
BSD-3-Clause LICENSE and third-party notices.

Inspect the executable's help before selecting an operation:

```sh
./vifstab --help
```

A caller with permission to install system software can copy it into an
appropriate directory, for example:

```sh
install -m 0755 vifstab /usr/local/sbin/vifstab
```

vifstab never acquires privileges or changes UID/GID. The caller supplies the
execution context. Do not install it setuid. Any configured editor and explicitly
selected eligible systemd reload inherit that same context.

Use `vifstab --help` to identify the installed interface. The verification-first source revision documents `verify`, `edit` and `format` separately. Older binaries may use different commands.

The tool is experimental. Keep independent configuration backups and read the [save-safety guide](docs/SAVE-SAFETY.md). Validation cannot guarantee bootability. Linux is the supported target; BSD and other fstab dialects are not supported.

## Verify the installed binary

```sh
vifstab --help
vifstab version
```

For release engineering and local snapshot builds, see [Contributing](CONTRIBUTING.md#development-and-release-checks).
