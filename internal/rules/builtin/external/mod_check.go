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

	// Try checking modules.builtin or modules.dep
	if checkModulesDepOrBuiltin(c, moduleName) || checkModulesDepOrBuiltin(c, modNameAlternative) {
		return nil
	}

	// If we couldn't confidently verify it, it might just be unloaded or we couldn't parse the files.
	// As per PR comment, avoid warning solely because it is unloaded.
	return nil
}

func getUnameR(c *rules.Cache) string {
	if c != nil {
		if v, ok := c.Get(rules.CacheKey("proc_version")); ok {
			if b, ok := v.([]byte); ok {
				return parseUnameR(string(b))
			}
		}
	}
	d, err := os.ReadFile("/proc/version")
	if err == nil {
		if c != nil {
			c.Set(rules.CacheKey("proc_version"), d)
		}
		return parseUnameR(string(d))
	}
	return ""
}

func parseUnameR(procVersion string) string {
	// "Linux version 5.15.0-1040-aws (buildd@lcy02-amd64-001) ..."
	parts := strings.Split(procVersion, " ")
	if len(parts) > 2 && parts[0] == "Linux" && parts[1] == "version" {
		return parts[2]
	}
	return ""
}

func checkModulesDepOrBuiltin(c *rules.Cache, moduleName string) bool {
	if moduleName == "" {
		return false
	}
	uname := getUnameR(c)
	if uname == "" {
		return false // Couldn't determine kernel version
	}

	// Check modules.builtin
	builtinPath := fmt.Sprintf("/lib/modules/%s/modules.builtin", uname)
	if data := readCachedFile(c, builtinPath); data != nil {
		if strings.Contains(string(data), "/"+moduleName+".ko") {
			return true
		}
	}

	// Check modules.dep
	depPath := fmt.Sprintf("/lib/modules/%s/modules.dep", uname)
	if data := readCachedFile(c, depPath); data != nil {
		if strings.Contains(string(data), "/"+moduleName+".ko:") {
			return true
		}
	}

	return false
}

func readCachedFile(c *rules.Cache, path string) []byte {
	if c != nil {
		if v, ok := c.Get(rules.CacheKey("file_content:" + path)); ok {
			if b, ok := v.([]byte); ok {
				return b
			}
		}
	}
	d, err := os.ReadFile(path)
	if err == nil {
		if c != nil {
			c.Set(rules.CacheKey("file_content:"+path), d)
		}
		return d
	}
	return nil
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
