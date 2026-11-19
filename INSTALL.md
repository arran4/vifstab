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

The verification-first CLI in #182 is required before recommending a first
binary download. The legacy default branch opens an editor on bare invocation;
this release-automation branch alone does not change that interface. Read the
README and the built executable's help for the selected source revision.

The tool is experimental. Keep independent configuration backups; save safety,
symlink/concurrency/extended metadata and RCS identity concerns are tracked in
#153–#156. Validation cannot guarantee bootability. Linux 7-series compatibility
is a development goal, not a blanket runtime guarantee. Fixtures exercise kernel
version values without claiming validation on every corresponding live kernel.
BSD and other fstab dialects are not supported.

## Local release preparation

No version number or publication is implied by a local snapshot:

```sh
python3 scripts/update-notices.py --check
goreleaser check
goreleaser release --snapshot --clean
```

Use a patched supported Go toolchain; distribution CI selects Go 1.26.x while
source compatibility checks retain the Go 1.25 module minimum. Dependency
security clearance also requires #181 (compression update) and review of the
current vulnerability database. Keep GoReleaser/Python/Go tool versions with
validation evidence. Snapshot runs do not publish a release or push/create tags.

Event semantics in the prepared workflow:

- Ordinary branch pushes and PRs run their existing checks.
- A pushed `v*` tag runs checked release preparation without a manual tag job.
- Manual major/minor/patch modes retain the existing local-tag/release path.
- Manual test/rc/alpha modes retain GoReleaser snapshot semantics.
- `test-*` tag pushes run code checks without publishing.
- A GitHub `release: published` event runs code checks only, avoiding a second
  publication in response to GoReleaser creating a release.

Only the owner may select reviewed PRs, create/push a real release tag, publish
artifacts, change repository visibility or approve an announcement. This task
prepares those actions and performs none of them.
