# Metadata preservation contract for issue #155 — owner direction agreed, implementation pending

**Owner decision: 15 November 2026.** Default to a strict metadata-preservation
policy, with explicit, documented relaxed modes. This document defines the
behaviour to implement; it is **not** an implemented preservation guarantee.
Current saves replace the inode and preserve only ordinary permission bits;
ownership, special bits, ACLs, xattrs and security labels can be lost.
Issue #155 remains open. No default policy, CLI flag or privilege behaviour has
changed in this design-only PR.

## Implemented foundations

- [PR #184](https://github.com/arran4/vifstab/pull/184) supplies the shared effective
  regular target, including relative/absolute symlinks and parent-link handling.
- [PR #195](https://github.com/arran4/vifstab/pull/195) supplies a stable metadata
  reference and strict regular/no-follow descriptor opening before data access.
  Its shared helper also protects [PR #188](https://github.com/arran4/vifstab/pull/188)
  history/lock access. A future metadata provider should consume the already
  resolved target and checked descriptor, rather than repeat pathname discovery.
- [PR #185](https://github.com/arran4/vifstab/pull/185) supplies optimistic content/
  identity conflict detection and edited-content recovery. Metadata observations
  need coordination with those checks; shared files alone do not make every
  feature a mandatory linear stack.

These are working, tested path/descriptor foundations. Metadata capture/copy and
production save integration are still required under this agreed policy.

## Why a successful list is insufficient

Linux [listxattr(2)](https://man7.org/linux/man-pages/man2/listxattr.2.html) permits
inaccessible attribute names to be omitted even when listing succeeds. An empty
list proves neither that all namespaces were visible nor that no attributes
exist. [xattr(7)](https://man7.org/linux/man-pages/man7/xattr.7.html) describes
different access rules for user, trusted, system and security namespaces;
security-module policies can further constrain access. Root identity alone does
not prove complete visibility. A capability in a user namespace must not be
presented as proof of global access.

Observation must distinguish known values, explicitly supported empty values,
unavailable listing/reads and incomplete visibility. ENOTSUP/EACCES, a truncated
list or an attribute disappearing between list and get must not silently become
an empty metadata set. No getfattr/getfacl/setfattr/chattr subprocess, mounting,
credential switch or privilege acquisition belongs in this implementation.

## Owner-selected mode hierarchy

These modes are **complementary**, not competing design PRs. Exact spelling is
subject to CLI integration, but an interface like
`--metadata-policy=strict|supported|best-effort` is a candidate for explicit
`vifstab edit` and `format --in-place` paths.

| Mode | Intended contract | Failure behaviour |
|---|---|---|
| **Strict (default)** | Preserve and verify the defined *required supported* ownership, group, ordinary/special mode bits, ACL, xattr and security-label attributes of the original file. Never claim that an ordinary successful xattr enumeration proves global completeness. Explicitly state coverage and unknowns. | If required metadata is unavailable, cannot be safely reproduced or cannot be verified, refuse replacement before rename; retain a recoverable edited copy and explain the exact unknown or loss risk. Never silently downgrade. |
| **Preserve supported** | Reproduce and verify all metadata actually visible to the caller in the documented supported classes. State that hidden/unlisted metadata cannot be guaranteed. | Fail on any supported *observed* copy/verification failure; allow a deliberate user choice where the only limitation is unprovable completeness, with a clear diagnostic. |
| **Best effort (explicit opt-in)** | Preserve what the process can, enumerate what could not be copied/verified, and report loss/uncertainty instead of implying success. | Warn with specifics; any irrecoverable corruption of the destination or required safety barriers still refuses save. No invisible automatic fallback. |

A successful listing or UID 0 alone does not prove global visibility. Strict
mode must **not** pretend it has stronger evidence than the kernel supplies.
If a defined strict requirement cannot be met in a particular environment,
the tool should provide an actionable explanation and an *explicit* choice of a
less strict mode rather than simply disabling editor use without explanation.
No mode can mount, escalate privileges, clear immutable flags, silently strip
security controls or spawn external xattr/ACL helper tools.

**fstab target versus RCS history:** Apply strict by default to the actual
fstab target, especially `/etc/fstab`. A backup archive may have a less
restrictive **inode metadata preservation** policy, with a warning and an
optional strict policy; it must *still* preserve content integrity, identity,
collision isolation, revisions, atomicity and recoverability. Do not conflate
archive metadata policy with permission to lose revision history.

**Independent verification rule:** [issue #202](https://github.com/arran4/vifstab/issues/202)
tracks a read-only advisory for suspicious file ownership/mode/ACL write
exposure. It does not change permissions or replace save-time preservation.

## Transaction common to all supported policies

1. Reuse the effective target; observe metadata through an injected editor
   provider on a checked regular descriptor. Record observation scope and file
   identity. No source absence is inferred from unavailable access.
2. Write edited content into the existing private temporary inode in the resolved
   target directory. Keep the established new-file policy separate from copying
   metadata from an existing file.
3. Apply ownership using caller permissions, then required mode bits and the
   supported attributes/ACL/security metadata in an order that preserves their
   final values. File chown operations do not change process credentials.
4. Check for inherited destination attributes that conflict with the selected
   source contract; do not silently leave a different security label or ACL.
   Preserve zero-length attribute values separately from absent attributes.
5. Re-observe and verify the required uid/gid, special/ordinary mode bits and
   attribute bytes on the temporary inode. Chown can clear set-id/capability
   metadata, ACL changes can affect mode masks, and
   [chmod(2)](https://man7.org/linux/man-pages/man2/chmod.2.html) can silently clear
   a set-group-ID bit in some contexts. A syscall success alone is not proof.
6. Under strict mode, any required unknown/copy/verification failure stops before replacement; reduced modes retain their explicitly defined refusal and warning rules. Close
   and remove incomplete destination temporaries; retain edited-content recovery
   through the editor workflow. Do not create a revision or run systemctl.
7. Sync data and metadata, close, recheck the selected source observations and
   conflict state, rename atomically and sync the containing directory. Preserve
   existing RCS/post-save error semantics. Rechecking is optimistic; it does not
   make compare-and-rename race-free against hostile writers.

The supported metadata set must be explicit. Do not claim all inode properties
or every security module/filesystem is covered merely because POSIX ACL xattrs
can be copied. Immutable/read-only barriers and unsupported metadata remain real
errors. Never clear protective flags or retry with greater privileges to save.

## Acceptance fixtures for implementation

| Fixture | Required result |
|---|---|
| Different source uid/gid; caller can reproduce them | Correct final ownership, with no process credential change |
| Ownership/mode operation denied | Original intact; edited content recoverable; no rename/revision/reload |
| Special bit silently cleared after an apparently successful chmod | Verification detects the mismatch before replacement |
| Binary/empty xattr values and ACL/security attributes | Exact supported values preserved according to the selected contract |
| Extra inherited target ACL/label | Explicitly reconcile or refuse; no silent incompatible result |
| Empty successful list with unknown global visibility | Unknown completeness; apply the selected contract without inventing absence |
| Listing/get/set/remove error, malformed/truncated list or changing source | Required failure before replacement; original remains intact |
| Symlink target, broken/looping links, retargeting and concurrent source change | Reuse effective-target/conflict behavior; no device data open |
| Sync/close/rename/directory-sync/RCS faults | Preserve existing atomicity, durability, recovery and post-save distinctions |

Use deterministic providers/TXTAR cases and temporary regular files; native
feature-dependent tests must report unavailable observations honestly. Never use
host fstab, real devices or a mounted filesystem as a content verifier.

This issue materially limits editor safety claims. It does not make every
optional feature a prerequisite for audited source visibility or for a separately
scoped read-only tester build. No production-safe editing claim is justified.
