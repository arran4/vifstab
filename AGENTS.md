# Agent Guidance

This repository implements a safety-oriented, visudo-like workflow for editing and validating `fstab`. Prefer conservative results, deterministic tests, and existing abstractions over shortcuts that depend on the host system.

## Scope and safety constraints

* Do **not** execute or depend on external block/system discovery commands such as `lsblk`, `blkid`, `udevadm`, `fdisk`, `efibootmgr`, `bootctl`, etc. Product logic should model the underlying Linux/UEFI interfaces directly.
* Do **not** open/read block devices directly in the normal/default mode.
* Live block-device metadata must come from normal filesystem/kernel metadata interfaces such as directory enumeration, symlinks, `stat`/`fstat`-style information, `/sys`, `/run/udev/data`, `/dev/disk/by-*`, `/proc`, and EFI variables where appropriate.
* Verifying filesystem **contents** on a block device is currently **out of scope**. Do not mount a device, require it to be mounted, or rely on the OS/VFS view of a mounted filesystem in order to validate its contents.
* It is acceptable to report information about an already-mounted filesystem when that information is optional and failure/unavailability is represented as **unknown**, not as a validation failure.
* A future explicitly enabled mode may inspect filesystem contents, but it should do so by reading the block device directly and parsing the filesystem format itself rather than mounting it or relying on the host OS to interpret the filesystem.
* A future mode may also explicitly opt in to reading block devices directly to inspect partition tables/filesystem headers, but ONLY through an explicitly enabled future mode. Do not implement that mode unless the task explicitly calls for it.
* `vifstab` does not acquire or manage privileges, switch users, or implement sudo/su/setuid/runas behaviour. Preserve the caller-supplied execution context.

## Existing architecture to reuse

* `internal/rules/api.go` is the canonical live-system abstraction. Reuse `rules.API`, `rules.ActualFS`, `OSProvider`, `BlockDeviceInventory`, and `BlockDeviceInventory.Resolve` instead of adding parallel device-discovery logic.
* Treat the block-device inventory as enrichment from several kernel/filesystem sources. Missing metadata is not automatically proof that a device or property is absent.
* `internal/rules/` contains the validation framework and registry. Put reusable validation policy there rather than embedding new lint policy directly in the CLI.
* `internal/rules/builtin`, `core`, `kernel`, and `options` contain existing rule families; inspect nearby rules and tests before creating a new pattern.
* `internal/editor/` owns the edit/verify workflow.
* `internal/format/` owns fstab formatting.
* `internal/diagram/` owns block-device diagram output.
* `internal/uefi/` owns UEFI/efivar parsing and correlation. Keep firmware parsing independent of systemd and external EFI tools.
* `internal/systemd/` is specifically for the optional post-save systemd integration; do not make unrelated validation depend on systemd.
* `internal/rcs/` owns backup/versioning behaviour.
* CLI entry points and user-visible subcommand wiring live under `cmd/vifstab/`.

## Unknown versus false

Live verification must distinguish **unknown/unavailable** from a definite negative result.

Examples:

* permission denied reading metadata -> unknown, not absent;
* unreadable/malformed EFI variables -> unknown, not "no matching boot entry";
* an unmounted device -> filesystem contents unknown, not empty;
* incomplete live inventory -> preserve diagnostics/unknown state rather than inventing certainty.

Only emit a high-confidence mismatch/absence after the relevant data was successfully obtained and structurally validated.

## Testability

* Host-dependent code must be behind an injectable provider/interface. Do not write tests whose result depends on the machine running `go test`.
* Prefer fixture-based tests for `/sys`, `/proc`, `/run/udev`, symlink resolution, EFI variables, and other live metadata.
* For binary formats such as EFI device paths, construct structurally valid fixtures and test bounds/truncation/malformed cases as well as the happy path.
* Reuse the canonical resolver in tests; do not make production and test resolution rules diverge.
* When adding a bug fix, add the smallest regression test that would have failed before the fix.

## Fast validation workflow

The module targets Go 1.25.

For a focused change, start with the affected package, for example:

```sh
go test ./internal/uefi/...
go test ./internal/rules/...
go test ./internal/editor/...
```

Then run the repository-wide checks before considering the work complete:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

If repository CI fails before Go jobs start (for example workflow routing/account infrastructure), distinguish that from a code/test failure instead of changing application code to satisfy an unrelated CI problem.

## User-visible changes

* When CLI syntax, flags, or subcommands change, keep `README.md` and `cmd/vifstab/manpage.md` in sync with the implementation.
* Keep diagnostics conservative and explicit. Prefer "unknown" over a misleading definitive answer.
* Do not silently add privilege escalation, mounting, destructive disk operations, or host-specific command dependencies.

## Working on issues and PRs

* Read the issue acceptance criteria and nearby tests before changing code; avoid solving a broader problem than requested.
* Preserve existing abstractions and behavioural contracts unless the issue specifically requires changing them.
* Keep changes focused; avoid unrelated refactors while resolving a bug/feature request.
* Use resolving keywords only when the implementation actually satisfies the issue's intended acceptance criteria. If a PR is only a foundation for later integration, leave the broader issue open.
* Before declaring work complete, verify that the pushed branch actually contains the claimed changes and that regression tests cover the reviewed failure mode.
