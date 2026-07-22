package external

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

func CheckModuleSupport(ctx context.Context, fsType string, entry *rules.FstabEntry, moduleName string, modNameAlternative string) []rules.ValidationIssue {
	c := rules.CacheFromContext(ctx)

	// First try to check /sys/module/<modulename>
	if checkSysModule(c, moduleName) || checkSysModule(c, modNameAlternative) {
		return nil
	}

	// Then check /proc/modules
	if checkProcModules(c, moduleName) || checkProcModules(c, modNameAlternative) {
		return nil
	}

	// Maybe checking config etc?
	// It's mostly a warning because they could be loadable modules not currently loaded,
	// or built out of tree.
	// Actually we should also check /lib/modules/$(uname -r)/modules.dep but we don't know uname -r exactly unless we check /proc/version.

	return []rules.ValidationIssue{{
		Message: fmt.Sprintf("Filesystem '%s' module '%s' does not appear to be loaded or built-in, mounting might fail on line %d", fsType, moduleName, entry.LineNum),
		Line:    entry.RawLine,
		Level:   rules.IssueLevelWarning,
	}}
}

func checkSysModule(c *rules.Cache, moduleName string) bool {
	if moduleName == "" {
		return false
	}
	path := fmt.Sprintf("/sys/module/%s", moduleName)
	if c != nil {
		if v, ok := c.Get(rules.CacheKey("file_exists:" + path)); ok {
			if exists, ok := v.(bool); ok {
				return exists
			}
		}
	}
	_, err := os.Stat(path)
	return err == nil
}

func checkProcModules(c *rules.Cache, moduleName string) bool {
	if moduleName == "" {
		return false
	}
	var data []byte
	if c != nil {
		if v, ok := c.Get(rules.CacheKey("proc_modules")); ok {
			if b, ok := v.([]byte); ok {
				data = b
			}
		}
	}

	if data == nil {
		d, err := os.ReadFile("/proc/modules")
		if err == nil {
			data = d
			if c != nil {
				c.Set(rules.CacheKey("proc_modules"), data)
			}
		}
	}

	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, moduleName+" ") {
			return true
		}
	}
	return false
}
