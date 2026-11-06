# Offline systemd generator-option checks

The shared `systemd-options` rule validates `x-systemd.*` independently of native
filesystem rules. It also covers swap, bind and unknown filesystem types. Native
option handlers delegate this namespace, so each finding appears once. The rule
uses the ordered `OptionSet`; it never normalises or rewrites options and visits
systemd occurrences in source order. Different rule families retain their own
normal reporting order.

These checks need no daemon, systemd installation, mountpoint, mounted filesystem
or block-device access. They do not run `systemctl` or alter process credentials.
Optional post-save daemon reload remains a separate explicit editing operation;
a reload is not a remount.

## What is checked

| Family | Offline check |
| --- | --- |
| automount, makefs, growfs, rw-only, pcrfs, validatefs | Bare flags; reject `=value` and `=` |
| idle-timeout, device-timeout, mount-timeout | Nonempty nonnegative systemd time span |
| requires, wants, before, after, wanted-by, required-by, onfailure | Nonempty unit/path dependency; repetitions retained |
| requires-mounts-for, wants-mounts-for | Nonempty absolute path; repetitions retained |
| device-bound | Bare option or supported case-insensitive boolean |
| graceful-option | Nonempty kernel-option shape; repeatable; exclude known userspace controls |

Time spans include decimal seconds, fractions, compound units such as `1h30min`,
standard microsecond spellings, `infinity`, and standard fstab space escapes such
as `1h\04030min`. Arithmetic checks the unsigned microsecond domain rather than
Go's smaller `time.Duration` domain. Zero is accepted; the inspected generator
maps zero timeouts to infinity. Negative and overflowing spans are errors.
Quoted or remaining escaped duration forms are explicitly **unverified** rather
than claimed equivalent to the documented unquoted syntax. Source spelling stays
unchanged.

Dependency checks accept documented unit names and absolute device/mount paths.
Upstream also mangles shorthand or other strings. Such spellings receive
nonblocking unverified guidance instead of an invented universal rejection.
There is no unit-existence, path-existence, remote-host, kernel-support or
filesystem-content check. Accepting `graceful-option` syntax does not establish
that its nested option is supported by the target kernel or filesystem.

## Diagnostic policy

Malformed known syntax is an error with line, key, reason and correction.
Unknown/newer names stay nonblocking information and explicitly unverified;
`X-systemd.*` stays a distinct generic `X-` namespace.

Repeated dependency/path/graceful options keep all values. Repeated singleton
settings receive one informational review note; legitimate overriding syntax is
not turned into a new error, and each earlier value is still checked. Existing
native filesystem conflict policies are unchanged.

`auto` and `noauto` do not disable a requested ordinary automount. An idle timeout
without automount gets nonblocking guidance. Legacy `comment=systemd.automount`
is recognised for that context. Mandatory root/usr handling and swap limitations
are reported conservatively; ordinary on-demand semantics are not promised for
those entries. `makefs` calls attention to device initialization, without
examining whether a device is empty or performing an operation. `_netdev`,
`nofail` and initrd behavior are not inferred from syntax or live mount state.

## Versions and limits

Normal CLI verification has **no automatic systemd-version detector**. An absent,
malformed or unavailable observation produces nonblocking unknown information
once per entry containing known systemd options. A distro version, PID1 name or
executable filename is not used as a substitute. This does not claim systemd is
installed, absent or the active init system.

Integrations/fixtures can supply `options.TargetSystemdVersion` (positive Major)
in the existing rules cache at `options.SystemdVersionKey`. Decimal fixture
strings are also supported. The value identifies an explicitly supplied target,
which may differ from the host. No new CLI option or host probe is introduced.
An option whose documented introduction is newer than a trustworthy supplied
target generates a version warning. Warnings follow the existing blocking policy;
unknown observations never become unsupported-version warnings.

The catalogue was checked against the upstream
[systemd.mount source at ec106549](https://github.com/systemd/systemd/blob/ec1065495802b32b42728914c29cc7252a9910fb/man/systemd.mount.xml),
[generator](https://github.com/systemd/systemd/blob/ec1065495802b32b42728914c29cc7252a9910fb/src/fstab-generator/fstab-generator.c),
and time/unit parsing sources. Newer catalogue entries include wants (257),
graceful-option and validatefs (258), and onfailure (262). Historical manuals show
wants-mounts-for absent in
[v255](https://github.com/systemd/systemd/blob/v255/man/systemd.mount.xml) and present
in [v256](https://github.com/systemd/systemd/blob/v256/man/systemd.mount.xml); the
current manual's combined v220 note is not applied to both path-option names.
This is a versioned, deliberately bounded catalogue, not complete emulation of a
particular generator, installed helpers, optional build features or initrd context.
The Go implementation introduces no runtime dependency on upstream systemd code.

## Reproducible cases

```sh
go test ./internal/rules/options/... -run 'TestSystemd'
go test ./internal/rules/builtin -run '^TestAllTxtarRules$/^testdata$/^systemd_options_'
go test ./internal/rules/builtin -run '^TestSystemd'
```

Five TXTAR fixtures exercise syntax, ordinary/EFI automount examples, context,
repetition, supplied old/unknown versions and escaped values. Their option fields
also round-trip through formatting. A complete-registry test covers sixteen
filesystem/dialect names and proves systemd diagnostics occur once, including
swap/bind/unknown types. Unit tests cover all catalogue shapes, documented version
boundaries, invalid/unknown version observations, duration overflow and malformed
input. Fuzz seeds exercise parser bounds without systemd or host data.
