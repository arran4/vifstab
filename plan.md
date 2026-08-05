1. **Understand the Goal**: The task requires adding `lustre` file system mount options to `vifstab` rules, based on the provided documentation link (https://doc.lustre.org/lustre_manual.xhtml#mountingclient).
2. **Review the existing code**: The file `internal/rules/builtin/external/lustre/lustre.go` currently only checks for module support. It should also validate the `lustre` mount options using the `options` package.
3. **Identify Lustre Mount Options**: Based on the documentation (`mount.lustre` section, table 44.13.3) and previous trials, the options are:
    - Server options: `abort_recov`, `nosvc`, `nomgs`, `exclude`, `md_stripe_cache_size`, `max_sectors_kb`, `recovery_time_soft`, `recovery_time_hard`, `noscrub`
    - Client options: `flock`, `noflock`, `localflock`, `user_xattr`, `nouser_xattr`, `user_fid2path`, `nouser_fid2path`, `retry=`, `always_ping`, `lazystatfs`, `nolazystatfs`, `verbose`, `noverbose`, `mgssec=` (flavors: `null`, `plain`, `gssnull`, `krb5n`, `krb5a`, `krb5i`, `krb5p`, `skn`, `ska`, `ski`, `skpi`), `skpath=`, `mgsname=`, `mgsnode=`
4. **Implement Validations**:
    - Update `lustreOptions` map in `internal/rules/builtin/external/lustre/lustre.go` with the appropriate `options.*Option{}` instances.
    - Call `options.ValidateOptionsMap` inside the `Validate` function of `LustreRule`.
5. **Update Test Cases**:
    - Ensure `internal/rules/builtin/testdata/fs_lustre.txtar` covers all valid and invalid cases for these options.
6. **Execute Pre-commit Checks**: Run tests and linting to make sure everything works perfectly.
