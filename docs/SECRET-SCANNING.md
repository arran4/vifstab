# Secret scanning and pre-publication checks

`vifstab` is intended to become public while it is in a **verification and
expansion** stage. The primary confidentiality risks are actual passwords,
tokens, private keys, non-public server addresses and proprietary system data.
Real filesystem UUIDs and other historical identifiers are not automatically
secrets. Prefer realistic synthetic identities for new public fixtures, without
unnecessary history rewrites.

## Continuous checking

`.github/workflows/gitleaks.yml` runs on every pull request (including PRs
targeting feature branches), pushes to `main` and manual dispatch. It uses
SHA-pinned, Node 24 compatible checkout/Gitleaks Actions and a read-only
`GITHUB_TOKEN`; comments, SARIF/report artifacts and job summaries are
disabled to reduce accidental publication of findings. There are no application
privileges, credentials or custom organization secrets involved.

Gitleaks identifies common secret formats. It **does not** establish that
every internal hostname, personal name, private mount path or custom password
has been found, and it does not certify that captures are anonymous.

A job that never gets a runner, fails to start or is skipped **has not passed**.
The existing private-repository runner-admission problem should not be treated
as an application or scanner result. A clean run after publication is useful,
but cannot replace the pre-publication audit.

## One-time audit before changing visibility

Use a **trusted, local, complete clone**, not an untrusted PR checkout. Do not
upload raw reports, verbose finding output, credentials, or captured archives to
issues or artifacts. With a current installed `gitleaks` CLI:

```sh
git fetch --all --tags --prune
# Include GitHub PR-head refs where accessible; the normal branch fetch
# may not contain every PR head.
git fetch origin '+refs/pull/*/head:refs/remotes/origin/pr/*'
gitleaks git --redact --log-opts="--all" .
gitleaks dir --redact .
```

Run these commands from the full clone with access to all intended refs.
The `git` check covers Git commit patches; the `dir` check also scans
working-tree files. In addition, manually inspect historically private
fixtures, deleted-file history, TXTAR captures, CI configuration, README,
issue/PR text and relevant GitHub Actions outputs to the extent available.

When a scanner finds a potential leak:

1. Review the finding **privately**, without including its secret value in a
   public comment. Check whether it is a known synthetic fixture or a real
   credential/private endpoint.
2. If it is a credential, revoke/rotate it; deleting a current file alone
   does not remove it from history.
3. Decide whether an owner-approved history rewrite is necessary. Do **not**
   rewrite the PR forest or remove branches just for harmless UUIDs.
4. Rerun the all-ref scan and record *redacted* evidence, scope, scanner
   version and remaining limitations privately.
5. Independently check private server addresses, usernames and configurations
   that pattern-based scanning may miss.

Gitleaks failures should block publishing until reviewed. If the scan has not
run on the complete intended history, mark exposure clearance **unverified**,
not clean. Repository visibility change, PR merges and release publication are
separate explicit owner actions.

Related: [public contribution hardening #186](https://github.com/arran4/vifstab/issues/186),
[TXTAR privacy and fixture policy #168](https://github.com/arran4/vifstab/issues/168),
[secret-scan tracking #203](https://github.com/arran4/vifstab/issues/203).
