package ext4

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "ext4", ext4Options), nil
}

var ext4Options = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"acl":          rules.NoValueOption{},
	"noacl":        rules.NoValueOption{},
	"bsddf":        rules.NoValueOption{},
	"minixdf":      rules.NoValueOption{},
	"check":        rules.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext4 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
	"nocheck":      rules.NoValueOption{},
	"debug":        rules.NoValueOption{},
	"errors":       rules.EnumOption{Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true}},
	"grpid":        rules.NoValueOption{},
	"bsdgroups":    rules.NoValueOption{},
	"nogrpid":      rules.NoValueOption{},
	"sysvgroups":   rules.NoValueOption{},
	"grpquota":     rules.NoValueOption{},
	"usrquota":     rules.NoValueOption{},
	"quota":        rules.NoValueOption{},
	"noquota":      rules.NoValueOption{},
	"nouid32":      rules.NoValueOption{},
	"oldalloc":     rules.NoValueOption{},
	"orlov":        rules.NoValueOption{},
	"resgid":       rules.IntOption{},
	"resuid":       rules.IntOption{},
	"sb":           rules.IntOption{},
	"user_xattr":   rules.NoValueOption{},
	"nouser_xattr": rules.NoValueOption{},

	"journal_dev":  rules.IntOption{},
	"journal_path": rules.StringOption{},
	"norecovery":   rules.NoValueOption{},
	"noload":       rules.NoValueOption{},
	"data":         rules.EnumOption{Allowed: map[string]bool{"journal": true, "ordered": true, "writeback": true}},
	"data_err":     rules.EnumOption{Allowed: map[string]bool{"abort": true, "ignore": true}},
	"barrier":      rules.EnumOption{Allowed: map[string]bool{"0": true, "1": true}},
	"commit":       rules.IntOption{},
	"jqfmt":        rules.EnumOption{Allowed: map[string]bool{"vfsold": true, "vfsv0": true, "vfsv1": true}},
	"usrjquota":    rules.StringOption{},
	"grpjquota":    rules.StringOption{},

	"journal_checksum":     rules.NoValueOption{},
	"nojournal_checksum":   rules.NoValueOption{},
	"journal_async_commit": rules.NoValueOption{},
	"nobarrier":            rules.NoValueOption{},
	"inode_readahead_blks": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"stripe":          rules.IntOption{},
	"delalloc":        rules.NoValueOption{},
	"nodelalloc":      rules.NoValueOption{},
	"max_batch_time":  rules.IntOption{},
	"min_batch_time":  rules.IntOption{},
	"journal_ioprio": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"abort":            rules.NoValueOption{},
	"auto_da_alloc":    rules.NoValueOption{},
	"noauto_da_alloc":  rules.NoValueOption{},
	"noinit_itable":    rules.NoValueOption{},
	"init_itable":      rules.IntOption{},
	"discard":          rules.NoValueOption{},
	"nodiscard":        rules.NoValueOption{},
	"block_validity":   rules.NoValueOption{},
	"noblock_validity": rules.NoValueOption{},
	"dioread_lock":     rules.NoValueOption{},
	"dioread_nolock":   rules.NoValueOption{},
	"max_dir_size_kb":  rules.IntOption{},
	"i_version":        rules.NoValueOption{},
	"nombcache":        rules.NoValueOption{},
	"prjquota":         rules.NoValueOption{},
})
