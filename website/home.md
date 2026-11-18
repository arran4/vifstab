# vifstab

Check your Linux filesystem configuration with confidence: vifstab finds
configuration mistakes and reports what it can verify before you rely on an
fstab entry.

**Experimental Linux utility.** Validation cannot guarantee bootability;
keep independent backups of important configurations.

## Start with verification

Download the appropriate Linux binary from [GitHub Releases](https://github.com/arran4/vifstab/releases)
or follow the [installation guide](../INSTALL.md). On the verification-first CLI
described by this site, try a disposable configuration:

```sh
printf '# example only\n' > example.fstab
vifstab verify --static --file ./example.fstab
```

Static verification avoids host observations. Live verification uses available
metadata; it does not edit or mount filesystems. Older binaries may have
different commands; check the installed version's help.

## Explore the documentation

- [User guide](../docs/USER-GUIDE.md): verification, explicit editing, formatting and offline replay.
- [Command reference](../cmd/vifstab/manpage.md): syntax, defaults and exit codes.
- [Validation and limits](../docs/VALIDATION.md): what the checks can and cannot establish.
- [Save safety](../docs/SAVE-SAFETY.md): write-time checks and recovery boundaries.
- [Support and privacy](../docs/CAPTURE-PRIVACY.md): prepare a synthetic or reviewed support case.

This site documents the default-branch source; released binaries may lag.
[Browse the source](https://github.com/arran4/vifstab) or
[contribute a test case](../CONTRIBUTING.md). Licensed under
[BSD 3-Clause](../LICENSE).
