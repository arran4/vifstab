package ext4

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Ext4Rule struct{}

func init() {
	rules.Register(&Ext4Rule{})
}

func (r *Ext4Rule) Name() string {
	return "ext4"
}

func (r *Ext4Rule) Dependencies() []string {
	return []string{}
}

func (r *Ext4Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext4" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "ext4", ext4Options), nil
}

var ext4Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":          options.NoValueOption{},
	"noacl":        options.NoValueOption{},
	"bsddf":        options.NoValueOption{},
	"minixdf":      options.NoValueOption{},
	"check":        options.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext4 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
	"nocheck":      options.NoValueOption{},
	"debug":        options.NoValueOption{},
	"errors":       options.EnumOption{Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true}},
	"grpid":        options.NoValueOption{},
	"bsdgroups":    options.NoValueOption{},
	"nogrpid":      options.NoValueOption{},
	"sysvgroups":   options.NoValueOption{},
	"grpquota":     options.NoValueOption{},
	"usrquota":     options.NoValueOption{},
	"quota":        options.NoValueOption{},
	"noquota":      options.NoValueOption{},
	"nouid32":      options.NoValueOption{},
	"oldalloc":     options.NoValueOption{},
	"orlov":        options.NoValueOption{},
	"resgid":       options.IntOption{},
	"resuid":       options.IntOption{},
	"sb":           options.IntOption{},
	"user_xattr":   options.NoValueOption{},
	"nouser_xattr": options.NoValueOption{},

	"journal_dev":  options.IntOption{},
	"journal_path": options.TODOOption{Comment: "validate journal device path"},
	"norecovery":   options.NoValueOption{},
	"noload":       options.NoValueOption{},
	"data": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		issues := (options.EnumOption{Allowed: map[string]bool{"journal": true, "ordered": true, "writeback": true}}).
			ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
		if len(issues) != 0 || val != "journal" {
			return issues
		}
		// Only warn when data=journal is the effective (last) mode. Earlier
		// occurrences can be overridden by a later data=ordered/writeback.
		if last, ok := allOpts[key]; ok && last.Value != "journal" {
			return nil
		}

		// Linux 7.3 deprecates data=journal; upstream plans to remove it in
		// January 2028, but has not specified a kernel release for removal.
		// Do not turn that calendar target into a guessed kernel cutoff.
		if kernel.GetHostKernelVersion(api).AtLeast(7, 3, 0) {
			return []rules.ValidationIssue{{
				Message: fmt.Sprintf("ext4 mount option 'data=journal' is deprecated since kernel 7.3 on line %d; removal is planned for January 2028 (removal kernel version not yet confirmed). Consider the default 'data=ordered' after reviewing its different data-integrity guarantees", entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			}}
		}
		return nil
	}),
	"data_err":     options.EnumOption{Allowed: map[string]bool{"abort": true, "ignore": true}},
	"barrier":      options.EnumOption{Allowed: map[string]bool{"0": true, "1": true}},
	"commit":       options.IntOption{},
	"jqfmt":        options.EnumOption{Allowed: map[string]bool{"vfsold": true, "vfsv0": true, "vfsv1": true}},
	"usrjquota":    options.TODOOption{Comment: "validate user quota path"},
	"grpjquota":    options.TODOOption{Comment: "validate group quota path"},

	"journal_checksum":     options.NoValueOption{},
	"nojournal_checksum":   options.NoValueOption{},
	"journal_async_commit": options.NoValueOption{},
	"nobarrier":            options.NoValueOption{},
	"inode_readahead_blks": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		num, err := strconv.Atoi(val)
		if err != nil {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if num == 0 || (num&(num-1)) != 0 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%d' for ext4 mount option 'inode_readahead_blks' on line %d, must be a power of 2", num, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"stripe":         options.IntOption{},
	"delalloc":       options.NoValueOption{},
	"nodelalloc":     options.NoValueOption{},
	"max_batch_time": options.IntOption{},
	"min_batch_time": options.IntOption{},
	"journal_ioprio": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		num, err := strconv.Atoi(val)
		if err != nil {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if num < 0 || num > 7 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%d' for ext4 mount option 'journal_ioprio' on line %d, must be between 0 and 7", num, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"abort":            options.NoValueOption{},
	"auto_da_alloc":    options.NoValueOption{},
	"noauto_da_alloc":  options.NoValueOption{},
	"noinit_itable":    options.NoValueOption{},
	"init_itable":      options.IntOption{},
	"discard":          options.NoValueOption{},
	"nodiscard":        options.NoValueOption{},
	"block_validity":   options.NoValueOption{},
	"noblock_validity": options.NoValueOption{},
	"dioread_lock":     options.NoValueOption{},
	"dioread_nolock":   options.NoValueOption{},
	"max_dir_size_kb":  options.IntOption{},
	"i_version":        options.NoValueOption{},
	"nombcache":        options.NoValueOption{},
	"prjquota":         options.NoValueOption{},
})
