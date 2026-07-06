import re

def modify_file(filepath, search_replace_pairs):
    with open(filepath, 'r') as f:
        content = f.read()

    for search, replace in search_replace_pairs:
        if search in content:
            content = content.replace(search, replace)
        else:
            print(f"Could not find search string in {filepath}:\n{search}\n")

    with open(filepath, 'w') as f:
        f.write(content)

# fs.go
modify_file("internal/rules/builtin/fs.go", [
    (
"""func (r *FSRule) Validate(ctx context.Context, entry *rules.FstabEntry) []rules.Suggestion {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil
	}

	if len(entry.Fields) < 3 {
		return nil // Leave to format rule
	}

	fsType := entry.Fields[2]

	if fsType == "ignore" || fsType == "swap" || fsType == "none" || fsType == "bind" {
		return nil
	}

	c := rules.CacheFromContext(ctx)
	if c == nil {
		return nil // No cache, can't validate properly
	}

	val, ok := c.Get(ProcFilesystemsKey)
	if !ok {
		// Try to load it
		fsMap, err := LoadProcFilesystems("/proc/filesystems")
		if err != nil {
			return nil // Can't validate
		}
		c.Set(ProcFilesystemsKey, fsMap)
		val = fsMap
	}

	validFS, ok := val.(map[string]bool)
	if !ok {
		return nil
	}

	if !validFS[fsType] {
		return []rules.Suggestion{
			{
				Message: fmt.Sprintf("Unknown filesystem type '%s' on line %d", fsType, entry.LineNum),
				Line:    entry.RawLine,
			},
		}
	}

	return nil
}""",
"""func (r *FSRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.Suggestion, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}

	if len(entry.Fields) < 3 {
		return nil, nil // Leave to format rule
	}

	fsType := entry.Fields[2]

	if fsType == "ignore" || fsType == "swap" || fsType == "none" || fsType == "bind" {
		return nil, nil
	}

	c := rules.CacheFromContext(ctx)
	if c == nil {
		return nil, fmt.Errorf("no cache provided in context")
	}

	val, ok := c.Get(ProcFilesystemsKey)
	if !ok {
		// Try to load it
		fsMap, err := LoadProcFilesystems("/proc/filesystems")
		if err != nil {
			return nil, fmt.Errorf("failed to load proc filesystems: %w", err)
		}
		c.Set(ProcFilesystemsKey, fsMap)
		val = fsMap
	}

	validFS, ok := val.(map[string]bool)
	if !ok {
		return nil, fmt.Errorf("invalid cache type for proc_filesystems")
	}

	if !validFS[fsType] {
		return []rules.Suggestion{
			{
				Message: fmt.Sprintf("Unknown filesystem type '%s' on line %d", fsType, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.SuggestionLevelError,
			},
		}, nil
	}

	return nil, nil
}"""
    )
])

# runner_test.txtar
modify_file("internal/rules/testdata/runner.txtar", [
    (
"""-- input/etc/fstab --""",
"""test: format and linux_fs test

-- description.txt --
Ensures we correctly parse and lint fstab using the runner pattern.
-- input/etc/fstab --"""
    ),
    (
"""-- expected/output.txt --
Validation errors found:
 - Unknown filesystem type 'unknown' on line 4 (Line: /dev/sda3       /mnt            unknown defaults        0       2)""",
"""-- expected/output.txt --
Validation issues found:
 - [error] Unknown filesystem type 'unknown' on line 4 (Line: /dev/sda3       /mnt            unknown defaults        0       2)
"""
    )
])
