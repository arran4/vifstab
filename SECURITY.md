# Security policy

This project is under development. No released version or maintained security
support window is currently promised. Validation findings are advisory evidence,
not a guarantee that a machine will boot or a filesystem is safe. Editing and
in-place formatting still have save/concurrency/metadata/RCS limitations under
review; do not use them as a production-safety guarantee.

## Report privately

Do not post credentials, private support archives, exploit details or sensitive
system configuration in public issues, PRs, logs or discussions.

If GitHub shows **Report a vulnerability** under this repository's Security →
Advisories page, use that private channel. It is available only when the owner
has enabled private vulnerability reporting. If it is unavailable and you have
no established private contact with the maintainer, open a content-free issue
asking for a private security contact. Include no technical details, secrets,
attachments or affected private systems in that request; wait for a private
channel before sending the report. No security email address or response-time
commitment is implied by this policy.

In the private report, include the affected revision, a synthetic reproducer,
expected/actual behavior, impact and relevant test results. Minimize private data.
If a credential was exposed, arrange revocation/rotation with its owner; deleting
a current file does not remove Git history, cached logs or earlier downloads.
Coordinate disclosure with the maintainer rather than publishing an unreviewed
capture or fix containing confidential details.

## Boundaries

`vifstab` uses the caller's credentials and does not acquire privileges. Its
permitted process launches are the configured editor during an intentional edit
and the existing explicitly selected post-save systemd operation. A configured
editor is trusted caller-supplied code. Validation/replay must not launch tools,
mount filesystems or gain privileges. Unavailable observations must stay unknown.

For non-sensitive rule mistakes, use the normal bug template with a synthetic
fixture. For snapshot submissions, read [privacy guidance](docs/CAPTURE-PRIVACY.md).
Maintainer settings checks are tracked separately in
[workflow security](docs/WORKFLOW-SECURITY.md); this file does not assert that
private reporting or any repository setting has already been enabled.
