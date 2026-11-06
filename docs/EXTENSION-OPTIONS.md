# Extension mount options

The shared option registry distinguishes known value shapes from unverified
extensions. Unknown ordinary options remain errors. Unknown `x-`, `X-` and
`x-systemd.` keys produce **nonblocking information** stating that their syntax
and semantics were not checked. Future extensions and possible typos therefore
remain accepted without being presented as verified. Matching is case-sensitive.
No installed util-linux or systemd version is inferred.

The deliberately small generic catalogue checks:

| Option | Shape checked |
| --- | --- |
| `X-mount.mkdir[=mode]` | Optional octal value; an explicit empty value is invalid |
| `x-mount.mkdir[=mode]` | Same documented, deprecated alias |
| `X-mount.noloop` | Bare flag; `=value` and `=` are invalid |
| `X-mount.subdir=directory` | Required nonempty value; contents are never inspected |

Other extensions remain unverified. This includes lower-case forms for which no
alias is documented; the registry does not mechanically copy every `X-` key into
`x-`. Existing per-option flag validators also reject explicitly empty values
and cannot bypass their own value restriction because a key begins with `x-`.

Upstream util-linux distinguishes the namespaces: `X-*` carries application data
or comments that are not retained for subsequent operations or forwarded to mount
helpers/the kernel; `x-*` is retained as userspace data where libmount can maintain
it. Retention is not guaranteed across every operation. The lower-case mkdir
alias is deprecated since util-linux2.30. Source inspected:
[util-linux mount manual at bd3a77c](https://github.com/util-linux/util-linux/blob/bd3a77c82ba7d88235e2d8942b248e0f69488f2e/sys-utils/mount.8.adoc).

These are offline value-shape checks. They do not create directories, mount
anything, inspect a subdirectory, establish helper/kernel support, or verify the
privileges needed to apply an option. `X-mount.subdir` has upstream applicability
limits; accepting its string does not prove it will take effect.

The shared map entry point gives explicitly registered validators precedence.
The eight older filesystem rules with custom option loops delegate extension
handling to that same policy, retaining their existing native-option checks and
messages. Btrfs now uses the existing quote-aware `SplitOptions` as well.
Known systemd semantic rules remain tracked in #171. An explicitly registered
systemd validator takes precedence over the unverified namespace fallback; unknown
newer names remain nonblocking. `X-systemd.*` is an ordinary `X-` extension, not an
alias for systemd's lower-case namespace. The separate repeated-value API in #189
is not a dependency of this policy.

Run deterministic catalogue and actual-filesystem regression cases:

```sh
go test ./internal/rules/options/... -run 'TestExtensionPolicy|TestRegisteredSystemdValidator|TestNoValueOption'
go test ./internal/rules/builtin -run 'TestAllTxtarRules/.*/.*/testdata/extension_policy.txtar'
```

The twelve TXTAR fixtures cover ext4, vfat, tmpfs, NFS, XFS, Btrfs, f2fs, JFS,
nilfs2, romfs, cramfs and BFS. Each checks known valid shapes, malformed known
values and future extension diagnostics through that filesystem's actual rule.
