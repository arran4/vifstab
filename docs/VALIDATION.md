# Validation and observation limits

Validation distinguishes a successfully observed negative from missing or unreadable evidence. Static mode never substitutes the current host. See the [user guide](USER-GUIDE.md) and [command reference](../cmd/vifstab/manpage.md) for usage.

## Mount-option validation

The shared options API retains every occurrence in input order, including bare
`flag`, empty `flag=`, quoted commas and the escapes already handled by
`rules.SplitOptions`. It does not normalise, deduplicate or reorder source fields.
Formatting retains the original mount-option field.

`ParseOptionSet` provides `Ordered()`, `All(key)`, `Last(key)` and `Has(key)`.
Accessors return copies. `Last` is an explicit last-occurrence view, not a claim
about which option a particular mount helper or filesystem will apply. To examine
opposite flags such as `ro,rw` or `auto,noauto`, inspect `Ordered()` so their relative
order remains available. The representation itself does not reject repetitions or
legal overriding syntax.

Existing `OptionValidator` checks still run for every occurrence and receive the
legacy last-value map. Register `WithOptionSet` as the **outermost** map validator
to add a whole-entry `OptionSetValidator` check. Its optional per-occurrence
`Validator` can be any existing validator or wrapper. The whole-entry check runs
once for that key, at its first occurrence, with all earlier and later values
available. Keys are visited in source order, independent of Go map iteration.
Calling `ValidateOption` directly runs only the per-occurrence validator; use
`ValidateOptionsMap` to run both phases. Existing slice validators also use the
shared parser and receive separate copies of the ordered sequence.

A policy can now distinguish repeatable dependencies from singleton repetitions
and report both values without duplicating the same cross-option finding. This
foundation does not add systemd semantic rules, infer an installed systemd version
or change existing validators' conflict policies. Existing explicit
`ExclusiveGroup` checks still reject their configured conflicts; their policy is
separate from option storage. Unknown `x-` and `X-` extensions produce nonblocking unverified information;
acceptance does not mean their semantics were checked. The extension policy
policy is described in [extension validation](EXTENSION-OPTIONS.md). The dedicated
[systemd policy](SYSTEMD-OPTIONS.md) uses the ordered representation across all
filesystem types.

Run the deterministic API/TXTAR regression cases with:

```sh
go test ./internal/rules/options/... -run 'TestOrderedOptionsTXTAR|TestOptionSet|TestWithOptionSet|TestSliceValidators'
```

The three `ordered-*.txtar` fixtures exercise dependency repetition, a sample
singleton check, non-systemd values, source-order access, shape preservation and
format round trips. The sample singleton callback is test policy, not a registered
systemd rule. No host files, daemon, discovery command or filesystem contents are
needed.


## Account observations

UID/GID option validation has two separate inputs: numeric syntax/range, and an observation about account availability. Unavailable account data must not make an invalid numeric ID acceptable, or make a valid ID a definite missing account.

Linux numeric IDs range from `0` through `4294967294`. The all-ones `uid_t`/`gid_t` value is reserved as invalid. A value in range is not proof of a mapping in the caller's user namespace. Filesystem-specific enum alternatives are checked before interpreting the value as an ID. On an architecture where the existing provider's `int` argument cannot represent a valid ID, availability stays unknown without a wrapped lookup.

`ActualFS` and `CacheFS` use the same bounded local-file parser behind their existing injected `OSProvider`/`fs.FS` interfaces. A matching passwd record needs seven fields, a nonempty ordinary name, and valid numeric UID and primary GID; a group record needs four fields, a nonempty ordinary name and valid numeric GID. Comments, NIS compatibility directives, NUL-containing records, malformed/truncated records and invalid IDs do not establish presence. An unterminated final record is accepted at ordinary EOF only, never after a read fault. A complete matching record remains a positive observation even if another record is malformed or later data is unavailable.

Local-file EOF never establishes global absence. NSS can consult other sources, and an incomplete or empty file is not a complete account inventory. Open/read permission failures retain the permission-denied state; other unavailable or unmatched observations remain unknown. The option validators report both as information. This change adds no external account command, libc/NSS call, network query or live fallback to fixture providers.

Existing explicitly supplied cache observations (`ExistState` or legacy booleans) remain authoritative, including definite absence. Known-negative regression fixtures now declare these observations rather than deriving them from omitted local-file rows. Explicit negatives retain their existing diagnostic severity. This is an observation contract, not a new account-management backend.

Regression coverage exercises both providers, malformed and incomplete rows, permission and I/O faults, valid positives with unrelated corruption, explicit states, numeric boundaries, filesystem enum alternatives, and an end-to-end rule TXTAR containing positive, unknown, invalid and explicitly absent IDs. All account data comes from fixtures.

Primary references:

- [passwd(5)](https://man7.org/linux/man-pages/man5/passwd.5.html): seven-field local account records.
- [group(5)](https://man7.org/linux/man-pages/man5/group.5.html): four-field local group records.
- [nsswitch.conf(5)](https://man7.org/linux/man-pages/man5/nsswitch.conf.5.html): configurable sources for passwd and group databases.
- [Linux uidgid.h at f66469d9905b39d2c260a15d7d6db450f4db67c1](https://github.com/torvalds/linux/blob/f66469d9905b39d2c260a15d7d6db450f4db67c1/include/linux/uidgid.h): invalid-ID sentinels and validity predicates.



## Configuration text inputs

Fstab text inputs for verification, formatting and editing, and RCS archive text
reads, are classified before opening a data descriptor. Block-device inputs are
refused; unknown or inaccessible classification is an operational error. This
does not add device discovery or inspect any block-device contents.

Linux uses an `O_PATH` metadata reference, checks its type, and obtains read access
through that same inode under `/proc/self/fd`. A subsequent identity/type check
must also succeed. Retargeting the original pathname cannot redirect the data
open. Metadata and data descriptors use close-on-exec; the data descriptor is
read-only and ordinary permissions use the caller's credentials. There is no
unsafe pathname fallback when these interfaces are unavailable. See
[open(2)](https://man7.org/linux/man-pages/man2/open.2.html).

Regular files and symlinked files remain supported. Intentional named-pipe and
character-stream inputs retain blocking stream semantics. Formatting still
accepts piped stdin; its borrowed descriptor is checked before reading. A
caller-supplied block descriptor is refused as input. Reader-based parsing APIs
remain available for explicitly supplied data and deterministic tests; their
callers own that input. The helper does not close borrowed standard input.

The pathname helper currently requires Linux. Other platforms or unavailable
metadata/proc-fd interfaces report an explicit limitation. Linux remains the
supported product target. This does not implement BSD support or infer privileges
from the process identity.

This boundary does not guarantee a snapshot of concurrently modified file contents or prevent hostile directory changes. The separate [save path](SAVE-SAFETY.md) handles write-time checks. The image parser uses a separate regular-file guard; see [diagram sources](DIAGRAM-SOURCES.md).
