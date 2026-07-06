import re

def modify_file(filepath, search_replace_pairs):
    with open(filepath, 'r') as f:
        content = f.read()

    for search, replace in search_replace_pairs:
        if search in content:
            content = content.replace(search, replace)
        else:
            print(f"Could not find search string in {filepath}: {search}")

    with open(filepath, 'w') as f:
        f.write(content)

# editor.go
modify_file("internal/editor/editor.go", [
    (
"""import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rcs"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)""",
"""import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rcs"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)"""
    ),
    (
"""	tempFile, err := os.CreateTemp("", "vifstab-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())""",
"""	tempFile, err := os.CreateTemp("", "vifstab-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()"""
    ),
    (
"""	src, err := os.Open(cfg.FilePath)
	if err == nil {
		_, err = io.Copy(tempFile, src)
		src.Close()
		if err != nil {
			return fmt.Errorf("failed to copy to temp file: %w", err)
		}
	}
	tempFile.Close()""",
"""	src, err := os.Open(cfg.FilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to open source file: %w", err)
		}
	} else {
		_, err = io.Copy(tempFile, src)
		src.Close()
		if err != nil {
			return fmt.Errorf("failed to copy to temp file: %w", err)
		}
	}
	tempFile.Close()"""
    ),
    (
"""	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	for {
		// 2. Open Editor
		cmd := exec.Command(editor, tempFile.Name())""",
"""	editor := os.Getenv("EDITOR")
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		editorParts = []string{"vi"}
	}

	for {
		// 2. Open Editor
		cmd := exec.Command(editorParts[0], append(editorParts[1:], tempFile.Name())...)"""
    ),
    (
"""		fmt.Print("What now? (e)dit again, (c)ontinue and save anyway, (q)uit without saving: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		choice := scanner.Text()""",
"""		fmt.Print("What now? (e)dit again, (c)ontinue and save anyway, (q)uit without saving: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read user input: %w", scanner.Err())
		}
		choice := scanner.Text()"""
    ),
    (
"""	// Then copy to dest
	if err := os.WriteFile(cfg.FilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}""",
"""	// Then copy to dest atomically
	dir := filepath.Dir(cfg.FilePath)
	tmpDest, err := os.CreateTemp(dir, "vifstab-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary destination file: %w", err)
	}
	defer os.Remove(tmpDest.Name())

	if _, err := tmpDest.Write(content); err != nil {
		tmpDest.Close()
		return fmt.Errorf("failed to write to temporary destination file: %w", err)
	}
	if err := tmpDest.Close(); err != nil {
		return fmt.Errorf("failed to close temporary destination file: %w", err)
	}

	if err := os.Rename(tmpDest.Name(), cfg.FilePath); err != nil {
		return fmt.Errorf("failed to replace destination file: %w", err)
	}"""
    )
])

# rcs.go
modify_file("internal/rcs/rcs.go", [
    (
"""import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arran4/golang-rcs"
)""",
"""import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-rcs"
)"""
    ),
    (
"""	baseName := filePath[strings.LastIndex(filePath, "/")+1:]
	if baseName == filePath {
		baseName = filePath
	}
	rcsFilepath := config.Dir + "/" + baseName + ",v\"""",
"""	baseName := filepath.Base(filePath)
	rcsFilepath := filepath.Join(config.Dir, baseName+",v")"""
    ),
    (
"""			minor := headParts[1]
			newMinor := 0
			fmt.Sscanf(minor, "%d", &newMinor)
			newHead := fmt.Sprintf("%s.%d", headParts[0], newMinor+1)""",
"""			minor := headParts[1]
			newMinor, err := strconv.Atoi(minor)
			if err != nil {
				return fmt.Errorf("failed to parse minor revision %q: %w", minor, err)
			}
			newHead := fmt.Sprintf("%s.%d", headParts[0], newMinor+1)"""
    )
])

# format.go
modify_file("internal/rules/builtin/format.go", [
    (
"""	if strings.HasPrefix(entry.RawLine, "#") {
		return nil
	}""",
"""	if strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil
	}"""
    )
])

# fs.go
modify_file("internal/rules/builtin/fs.go", [
    (
"""	if len(entry.Fields) == 0 || strings.HasPrefix(entry.RawLine, "#") {
		return nil
	}""",
"""	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil
	}"""
    )
])
