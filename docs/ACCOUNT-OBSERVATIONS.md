# Account observation limits

UID/GID option validation has two separate inputs: numeric syntax/range, and an observation about account availability. Unavailable account data must not make an invalid numeric ID acceptable, or make a valid ID a definite missing account.

Linux numeric IDs range from `0` through `4294967294`. The all-ones `uid_t`/`gid_t` value is reserved as invalid. A value in range is not proof of a mapping in the caller's user namespace. Filesystem-specific enum alternatives are checked before interpreting the value as an ID. On an architecture where the existing provider's `int` argument cannot represent a valid ID, availability stays unknown without a wrapped lookup.

`ActualFS` and `CacheFS` use the same local-file parser behind their existing injected `OSProvider`/`fs.FS` interfaces. A matching passwd record needs seven fields, a nonempty ordinary name, and valid numeric UID and primary GID; a group record needs four fields, a nonempty ordinary name and valid numeric GID. Comments, NIS compatibility directives, NUL-containing records, malformed/truncated records and invalid IDs do not establish presence. An unterminated final record is accepted at ordinary EOF only, never after a read fault. Records are read without a fixed scanner token ceiling. A complete matching record remains a positive observation even if another record is malformed or later data is unavailable.

Local-file EOF never establishes global absence. NSS can consult other sources, and an incomplete or empty file is not a complete account inventory. Open/read permission failures retain the permission-denied state; other unavailable or unmatched observations remain unknown. The option validators report both as information. This change adds no external account command, libc/NSS call, network query or live fallback to fixture providers.

Existing explicitly supplied cache observations (`ExistState` or legacy booleans) remain authoritative, including definite absence. Known-negative regression fixtures now declare these observations rather than deriving them from omitted local-file rows. Explicit negatives retain their existing diagnostic severity. This is an observation contract, not a new account-management backend.

Regression coverage exercises both providers, malformed and incomplete rows, permission and I/O faults, valid positives with unrelated corruption, explicit states, numeric boundaries, filesystem enum alternatives, and an end-to-end rule TXTAR containing positive, unknown, invalid and explicitly absent IDs. All account data comes from fixtures.

Primary references:

- [passwd(5)](https://man7.org/linux/man-pages/man5/passwd.5.html): seven-field local account records.
- [group(5)](https://man7.org/linux/man-pages/man5/group.5.html): four-field local group records.
- [nsswitch.conf(5)](https://man7.org/linux/man-pages/man5/nsswitch.conf.5.html): configurable sources for passwd and group databases.
- [Linux uidgid.h at f66469d9905b39d2c260a15d7d6db450f4db67c1](https://github.com/torvalds/linux/blob/f66469d9905b39d2c260a15d7d6db450f4db67c1/include/linux/uidgid.h): invalid-ID sentinels and validity predicates.

Refs #198 and #157. This follow-up preserves the earlier open/read error handling and does not reopen the broader issue.
