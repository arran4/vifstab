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

# editor.go
modify_file("internal/editor/editor.go", [
    (
"""		fmt.Println("Validation errors found:")
		for _, s := range allSuggestions {
			fmt.Printf(" - %s (Line: %s)\\n", s.Message, s.Line)
		}""",
"""		fmt.Println("Validation issues found:")
		for _, s := range allSuggestions {
			fmt.Printf(" - [%s] %s (Line: %s)\\n", s.Level, s.Message, s.Line)
		}"""
    )
])

# runner_test.go
modify_file("internal/rules/runner_test.go", [
    (
"""			var allSuggestions []rules.Suggestion
			for _, entry := range entries {
				for _, r := range rulesToRun {
					suggestions := r.Validate(ctx, entry)
					allSuggestions = append(allSuggestions, suggestions...)
				}
			}

			// Construct got output
			var gotOutput string
			if len(allSuggestions) > 0 {
				gotOutput += "Validation errors found:\\n"
				for _, s := range allSuggestions {
					gotOutput += fmt.Sprintf(" - %s (Line: %s)\\n", s.Message, s.Line)
				}
			}""",
"""			var allSuggestions []rules.Suggestion
			var validateErr error
			for _, entry := range entries {
				for _, r := range rulesToRun {
					suggestions, err := r.Validate(ctx, entry)
					if err != nil {
						validateErr = fmt.Errorf("rule %s failed: %w", r.Name(), err)
						break
					}
					allSuggestions = append(allSuggestions, suggestions...)
				}
				if validateErr != nil {
					break
				}
			}

			// Construct got output
			var gotOutput string
			if validateErr != nil {
				gotOutput = fmt.Sprintf("Error: %v\\n", validateErr)
			} else if len(allSuggestions) > 0 {
				gotOutput += "Validation issues found:\\n"
				for _, s := range allSuggestions {
					gotOutput += fmt.Sprintf(" - [%s] %s (Line: %s)\\n", s.Level, s.Message, s.Line)
				}
			}"""
    )
])
