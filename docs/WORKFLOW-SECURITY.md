# Workflow permissions and maintainer checks

The workflow defaults to `contents: read`. Pull-request code checks use the
`pull_request` event, with no write-scoped token or retained checkout credentials.
There is no `pull_request_target` build and no custom repository secret is passed
to PR tests. Do not add privileged execution of untrusted head code, including
through downloaded artifacts or a follow-up workflow.

Only the explicit manual formatting job requests `contents: write` and
`pull-requests: write`. Its checkout retains credentials because it pushes the
formatting branch; its existing job condition requires manual dispatch,
`lint-fix` and explicit `allow_prs`. The tag/manual release job requests only
`contents: write` for GitHub releases. Neither write job accepts ordinary PR
events. Review the selected revision before invoking either automation path.
Discussions, packages, checks and security-events writes are not requested.

Actions are pinned to full commit SHAs resolved from their existing major tags.
The linter, GoReleaser and tag helper have explicit versions. Updating pins means
reviewing the upstream source/release and rerunning checks, not just accepting
a movable tag. The tag helper's composite action still downloads release assets;
version pinning alone is not independent binary provenance verification. The
legacy version-calculation fallback can also use npm's `npx semver`; review its
availability/supply chain before invoking manual release automation. Release
event/dependency repair is independent [PR #183](https://github.com/arran4/vifstab/pull/183),
and adoption of this permissions change alone does not fix release routing.

Before public contribution or release, the owner must inspect settings that
source cannot establish:

- Workflow token defaults, allowed Actions, fork-workflow approval and access to
  secrets/environments. Keep untrusted fork runs unprivileged and require
  appropriate human approval; do not enable write tokens or secrets for them.
- Protection/rulesets and required review/check policy on the default branch;
  do not count skipped or unassigned CI as successful verification.
- Private vulnerability reporting for the intended public repository. Confirm
  that contributors can use the private reporting channel in SECURITY.md.
- Historical refs, logs/artifacts, metadata, third-party notices and identifying
  data through the separate exposure audit. Source hardening is not audit clearance.

The current credentials cannot read all administration settings. This document
does not claim any setting has already been changed or verified. Visibility,
merges, tags, releases and announcements remain explicit owner actions.

References: [GitHub secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use),
[workflow permissions](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions),
and [private vulnerability reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository).
