1. Add the mount option parsing validation rules for `encfs` filesystem.
    - Update `internal/rules/builtin/layered/encfs/encfs.go` by:
        - Importing `"github.com/arran4/vifstab/internal/rules/options"`.
        - Calling `options.ValidateOptionsMap` in `Validate` method and returning its result instead of `nil, nil`.
        - Defining `encfsOptionsMap` with `options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{...})` map containing all options specified in the debian manpage for encfs.
2. Add `.txtar` tests to ensure the encfs options are properly validated.
    - Add `internal/rules/builtin/layered/encfs/testdata/encfs_ok.txtar` containing all valid encfs-specific options.
    - Add `internal/rules/builtin/layered/encfs/testdata/encfs_err.txtar` containing invalid encfs-specific options.
3. Complete pre commit steps to make sure proper testing, verifications, reviews and reflections are done.
4. Submit the change.
