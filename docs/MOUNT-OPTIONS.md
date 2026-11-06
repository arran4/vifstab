# Mount-option validation

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
acceptance does not mean their semantics were checked. The independent #172
policy is described in [extension validation](EXTENSION-OPTIONS.md). Issue #171
tracks dedicated systemd policy.

Run the deterministic API/TXTAR regression cases with:

```sh
go test ./internal/rules/options/... -run 'TestOrderedOptionsTXTAR|TestOptionSet|TestWithOptionSet|TestSliceValidators'
```

The three `ordered-*.txtar` fixtures exercise dependency repetition, a sample
singleton check, non-systemd values, source-order access, shape preservation and
format round trips. The sample singleton callback is test policy, not a registered
systemd rule. No host files, daemon, discovery command or filesystem contents are
needed.
