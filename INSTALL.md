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

## Container testing (Docker and Podman)

Once published, release images are available at
[ghcr.io/arran4/vifstab](https://github.com/arran4/vifstab/pkgs/container/vifstab)
for Linux amd64 and arm64. Use an existing release's exact `v...` tag (including
prereleases), or use `latest` once a stable image exists. For reproducible use,
pin the published manifest digest instead of a mutable tag. Do not assume an
image exists before its release workflow completes.

```sh
image=ghcr.io/arran4/vifstab:latest
docker run --rm --network none --cap-drop ALL "$image" --help
docker run --rm --network none --cap-drop ALL "$image" version
```

For a disposable synthetic configuration:

```sh
printf '# disposable example\n' > example.fstab
docker run --rm --network none --cap-drop ALL --read-only \
  -v "$PWD/example.fstab:/input.fstab:ro" \
  "$image" verify --static --file /input.fstab
printf '# example\n' | docker run --rm -i --network none \
  --cap-drop ALL --read-only "$image" format --file -
```

Capture and offline replay without mounting the host's system configuration:

```sh
(
  umask 077
  docker run --rm --network none --cap-drop ALL --read-only \
    -v "$PWD/example.fstab:/input.fstab:ro" \
    "$image" verify --static --file /input.fstab --capture - > case.txtar
)
docker run --rm -i --network none --cap-drop ALL --read-only \
  "$image" verify --snapshot - < case.txtar
```

The same commands work with `podman` substituted for `docker`. For rootless
Podman or SELinux, use a disposable fixture readable by the container identity;
stdin avoids host bind-mount permissions. Do not relax permissions on private
files. Review captured data before sharing.

The scratch image runs as numeric UID/GID 65532, with only the static executable
and licence notices. It has no shell or editor: interactive editing requires a
native installation. Live verification inside a container observes its own
namespace, not necessarily the host. Prefer static or snapshot modes. Never
mount host devices, the host root, writable system configuration or systemd
sockets, and do not use privileged containers for routine verification.

### Test a local image build

Maintainers can run the pinned GoReleaser v2.20.0 and Docker Buildx:

```sh
goreleaser check
goreleaser release --snapshot --clean
bash scripts/test-container-images.sh
```

This loads separate `snapshot-amd64` and `snapshot-arm64` local images; no
registry login or push. The manual release-test/rc/alpha workflow modes also
remain non-publishing snapshots. A pushed reviewed `v...` release tag
publishes a multi-platform image and native archives. Prereleases publish their
version tag, not `latest`. Stable releases publish both.

After the first owner-approved release, check the package is linked to this
repository and set its visibility to **Public** in GHCR package settings if
necessary. Test an anonymous pull with a fresh Docker config or Podman, and
inspect the manifest for both supported Linux platforms. The release workflow
checks the manifest with its scoped registry credential.
