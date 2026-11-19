# Optional drift and ESP checks — draft scope for #112/#113

This records narrow alternatives and deferral criteria. It implements no live
provider, rule, command or flag, and closes neither issue. Broad mount-status
reporting is outside the owner's current direction. Neither check gates static
verification, snapshot replay or source publication.

## Scope choices

| Issue | Narrow useful direction | Defer/reject |
|---|---|---|
| [112](https://github.com/arran4/vifstab/issues/112) | Explicitly selected observations of consequential configuration-versus-effective security options, after unambiguous source/mountpoint correlation | A findmnt replacement, automatic host inspection for arbitrary/foreign fstab files, or treating a valid entry as proof it is mounted |
| [113](https://github.com/arran4/vifstab/issues/113) | Explicitly scoped ESP permission findings using confident role metadata and understood FAT permission options | Inferring an EFI tree/boot-file contents, requiring mounts, treating missing metadata as not-an-ESP, or promising boot security from mount options |

The owner can defer both without blocking the implemented ordered options,
extension/systemd rules or contribution workflow. Any eventual live activation
must be an explicit, separate choice; no unfinished CLI flag is proposed as an
existing feature here.

## Observation boundary

Reuse rules.API/ActualFS/OSProvider and BlockDeviceInventory.Resolve. If new mount
observations are necessary, add a small optional provider interface backed by
normal kernel metadata such as the caller's proc mountinfo. Do not add a second
device-discovery path, external findmnt/mount/systemd/boot-tool commands, mounts,
privilege behavior or block-device reads. internal/systemd remains post-save
integration; reusable validation belongs under internal/rules.

State exactly which mount namespace and target configuration an observation
describes. A container/chroot view is not automatically the boot host's state.
Require successful bounded structural parsing, correct escaping, positive
correlation and sufficient relevant flags before a definite mismatch. An absent
correlation, an unmounted target, unreadable/malformed data or ambiguous identities
remains unknown. Unsupported option semantics remain unverified.

Static verification never needs these observations. Snapshot replay must use
only declared captured observations and never query the host. Capture schema v1
does not promise a complete effective-option view; do not silently extend it or
invent observations from its mountpoint labels. Any schema extension needs an
explicit version/compatibility/privacy design and bounds tests.

## Candidate findings, not approved policy

- Explicit configured read-only versus positively observed read/write at the
  same confidently identified source/mountpoint could be useful. Bind/remount,
  defaults, repeated flags, namespace differences and unknown semantics need
  separate handling; do not invent a universal last-value rule.
- A confidently identified ESP with explicitly broad writable FAT file/directory
  masks may justify an optional security finding. Validate umask/fmask/dmask,
  uid/gid and their actual filesystem semantics before claiming effective access.
  Missing masks or role data do not establish that permissions are safe or unsafe.
- Firmware variables can enrich role correlation only when structurally valid;
  malformed/unreadable variables stay unknown. UEFI parsing remains independent
  of systemd. No EFI tree, key or bootloader file content is inspected.

Severity/activation are owner choices. Current Warning/Security levels can block
verification; do not silently turn optional, uncertain observations into blocking
ordinary verification. Existing explicit read-only verification should remain
useful without a live mount view.

## Required evidence before an implementation is ready

Fixtures must cover positive/ambiguous source identities, mount namespaces,
octal escaping and separators, malformed/truncated records, unavailable reads,
unmounted cases, bind/remount/repeated-option ambiguity, FAT masks and missing
ESP metadata. Test all high-confidence findings and unknown alternatives through
the shared registry/provider path. Include static/snapshot no-host-access and
no-process/no-device-open assertions, exact diagnostics/statuses and unchanged
input files. Do not use host mount tables, EFI variables, fstab or devices.

If these narrow criteria cannot produce a useful, defensible finding, keep the
issues parked rather than adding general status output. No mount operation,
content verification, new announcement claim or publication schedule is approved.
