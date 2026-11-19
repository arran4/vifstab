# Diagram sources and limits

`vifstab diagram DEVICE_OR_IMAGE` is read-only. It keeps the caller's credentials
and uses no discovery commands, mount operations or privilege acquisition.

## Block-device references

Block-device arguments use the canonical `rules.ActualFS` inventory and
`BlockDeviceInventory.Resolve`, including recognised UUID/alias references.
Optional `IncludeTopology` enrichment reads normal kernel/udev metadata: sysfs
size/start/partition attributes, sysfs parent links/device identities, and an
available udev partition-table label. It does not open a block device for I/O or
parse its headers or filesystem contents.

The metadata view describes observed kernel ranges and udev labels. It does not
prove that an on-disk table is complete, current or valid. Metadata can be partial,
stale or unavailable; unusable topology fields stay unknown and diagnostics are
retained. A missing correlation is not proof that a device or partition is absent.
The existing inventory filters remain; unsupported or uncorrelated objects may
therefore have only an unknown diagram. No raw-read fallback is used.

`?` means unverified space, **not free space**. `*` means multiple observations
fall into the same approximate character cell. Missing/zero/inconsistent sizes
are not used for division or invented ranges. Filesystem-type labels are metadata
observations, not filesystem-content verification. Mountpoint information, where
available through the normal inventory, remains optional.

The Linux block layer expresses these size/start fields in 512-byte sectors,
independent of logical/physical block size. References:
[partition attributes](https://github.com/torvalds/linux/blob/f66469d9905b39d2c260a15d7d6db450f4db67c1/block/partitions/core.c),
[size attribute](https://github.com/torvalds/linux/blob/f66469d9905b39d2c260a15d7d6db450f4db67c1/block/genhd.c),
and [sector units](https://github.com/torvalds/linux/blob/f66469d9905b39d2c260a15d7d6db450f4db67c1/include/linux/blk_types.h).

Topology is an optional live inventory adjunct excluded from capture schema v1.
Ordinary verification/capture does not request these extra observations, and
this change does not add diagram replay or silently extend archived schemas.

## Regular disk images

Regular image files retain GPT/MBR partition-table parsing and the existing ASCII
format. Image parsing reads the supplied regular file; it is separate from
block-device metadata rendering. Symlinked regular images remain supported.

On Linux, an `O_PATH` metadata reference is checked before any data descriptor is
opened. A read-only descriptor is then obtained through that stable object under
`/proc/self/fd`, checked again for regular type/identity, and supplied to the
existing parser as a read-only backend without a pathname to reopen. Replacing
the original path with a device or FIFO cannot redirect that read. Both handles
use close-on-exec. A failed check/refused read remains an error; there is no
unsafe path-based fallback. See [open(2)](https://man7.org/linux/man-pages/man2/open.2.html).

This safe image opener currently requires Linux metadata/proc-fd interfaces;
unavailable interfaces or other platforms report the limitation. Linux is the
current platform target. Tests use injected kernel layouts and temporary regular
files, including a supplied root user-namespace context; they do not establish
compatibility with every kernel or promise future Linux7-series support.

Image parsing is not a filesystem-content validator or a guarantee against
concurrent changes to an image. Invalid/truncated tables, nonregular objects and
out-of-file partition bounds are refused. No live filesystem is mounted.

## Btrfs contents

The Btrfs subvolume option no longer consults the host VFS to infer contents of a
block filesystem. Unobserved contents stay unchecked/unknown. Explicit offline
fixture observations remain supported; an unrelated existing or missing host path
cannot establish whether that subvolume exists on the device.

## Fixture tests

```sh
go test ./internal/diagram/...
go test ./internal/rules/... -run 'TestBlockTopology'
go test ./internal/rules/builtin -run '^TestBtrfsSubvolumeContentsRemainUnknownWithoutObservation$'
go test ./cmd/vifstab -run '^TestDiagram$'
```

Metadata TXTAR cases run through the production inventory/resolver and cover
canonical paths, UUID aliases, unreadable sizes and missing parent correlations.
Other cases cover malformed/overflowing numbers, incomplete sources, identity
cycles, unavailable observations and short/writer errors. Image tests retain all
six legacy TXTARs and add real GPT/MBR files, symlinks, stable inode reopening after
path replacement, metadata/data descriptor flags, refusal before data-open and
failure cleanup. Only temporary fixture files are used.

Diagram output contains device paths and layout details. Review it before
including it in a public support report.
