# Privacy review for support cases

Treat an fstab, a TXTAR archive and command output as potentially private. Nothing
is uploaded automatically by vifstab. You decide what to share and where. A
public issue or PR can expose its text and attachments, including earlier edits.

Capture, static verification and offline replay are described in the [user guide](USER-GUIDE.md). Use the help and documentation for your exact revision; default redaction is not an anonymity guarantee.

Before sharing a case:

1. Work on a local copy. Inspect every archive section and all referenced text,
   including the fstab, observations, diagnostics and command output. Never
   attach an unreviewed original as an extra file or paste it into a log.
2. Remove credentials, passwords, tokens, mount `secret=`/`password=` values,
   credential files, key material and anything belonging to someone else.
3. Review UUID/PARTUUID/serial/WWN values, device aliases, labels, host/user/group
   names, network shares, IPs/domains, home paths, mount locations and firmware
   paths. Even ordinary identifiers can correlate a machine or person.
4. Replace identifying values consistently across the fstab, symlinks and all
   observations. Keep relationships needed to reproduce the bug; use invented
   values and document deliberate simplifications. Redaction can change semantics.
5. Remove unrelated sections and reduce to the smallest reproducible case. Run
   the available offline replay/fixture test for that revision and compare the
   result. Missing observations should remain unknown, not become false negatives.
6. Review the final bytes again before submitting. Redaction is not anonymity,
   and a successful scanner does not prove that a capture is safe to publish.

Prefer synthetic fixtures over real captures where possible. If faithful
reproduction requires confidential data, stop and arrange a private channel
through [SECURITY.md](../SECURITY.md). Do not publish the data to ask whether it
is sensitive. If you accidentally shared a secret, notify its owner privately,
rotate/revoke it where appropriate, and coordinate history/log remediation.

Do not mount, unmount, read raw block-device contents, run discovery commands or
escalate privileges to manufacture a support case. A device being unmounted or
unreadable means its contents are unknown. Review failure/unavailable states as
carefully as successful observations.
