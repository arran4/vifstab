# Website build and content inventory (maintainers)

This document is deliberately **not** published. Pages is configured once in
Settings → Pages → Build and deployment → Source: **GitHub Actions**. The
workflow builds on pushes to main/master and supports trusted default-branch
dispatch. PRs build and validate only; no publishing branch or generated HTML
is committed. After the first successful deployment, set the repository About
website field to the actual deployment URL. The expected URL is
https://arran4.github.io/vifstab/.

## Local build

Install Hugo 0.170.0 (the CI workflow verifies the release asset SHA-256) and
Python 3. From the repository root:

```sh
python3 website/build.py --output /tmp/vifstab-docs
```

Use a new/empty output directory. For a local preview at the domain root,
build with `--base-url http://localhost:8000/` and then serve the output using
`python3 -m http.server 8000 --directory /tmp/vifstab-docs`.
The default build checks the real project subpath. The build checks all local
HTML links/anchors and rejects unexpected output files, missing targets or
unlisted generated pages. External sites are not fetched by the link checker.

## Editorial inventory

The machine-readable `pages.json` is the only Markdown publication allowlist.
Site staging copies only those texts plus the local layout/CSS/config. Never
copy source trees or arbitrary files into the public directory.

| Source | Decision and rationale |
|---|---|
| README.md | Rewrite for discovery and quick start; website landing has independent short introduction and links. |
| INSTALL.md | Rewrite, publish installation guidance (move release engineering to contributor instructions). |
| cmd/vifstab/manpage.md | Retain authoritative command reference; publish. |
| docs/USER-GUIDE.md | New canonical guide extracted from the long README; publish. |
| docs/VALIDATION.md | Consolidate mount-option, account and text-input observations; publish. |
| docs/ACCOUNT-OBSERVATIONS.md, docs/MOUNT-OPTIONS.md, docs/TEXT-INPUTS.md | Remove redundant files after consolidation; preserve substantive contracts in VALIDATION.md. |
| docs/EXTENSION-OPTIONS.md, docs/SYSTEMD-OPTIONS.md | Retain separately useful detailed option compatibility reference; publish. |
| docs/SAVE-SAFETY.md, docs/RCS-HISTORIES.md | Retain materially useful recovery and migration boundaries; correct stale claims and publish. |
| docs/DIAGRAM-SOURCES.md | Retain detailed limits, useful for users of diagram; publish. |
| docs/CAPTURE-PRIVACY.md | Rewrite stale interface references; retain and publish. |
| CONTRIBUTING.md, SECURITY.md | Update stale claims; publish as contributor/security pages. |
| AGENTS.md, cmd/AGENTS.md | Retain architecture and generated-agent guidance; exclude. |
| docs/SECRET-SCANNING.md, docs/WORKFLOW-SECURITY.md | Retain active maintainer controls and audit procedure; exclude (repository-only). |
| THIRD_PARTY_NOTICES.md, LICENSE | Retain required distribution texts; link from README/site rather than reproduce notices as primary content. |
| Go sources, workflows, raw TXTAR fixtures, build products, handoff notes | Exclude completely. |
| website/BUILD.md | Maintainer workflow instructions; exclude. |
| website/home.md | Purpose-built editorial landing; publish. |

The site tracks the integrated source, not an assumed release. Update the
canonical manual and guides alongside CLI changes. Any new published page
requires an explicit inventory/allowlist change and a review.
