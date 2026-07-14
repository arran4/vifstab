package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func ListSkills(agent string, scope string, jsonOutput bool) error {
	tr := NewTargetResolver(agent, scope)
	basePath, err := tr.ResolvePath()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOutput {
				fmt.Println("[]")
			} else {
				fmt.Println("No skills installed.")
			}
			return nil
		}
		return err
	}

	var skills []SkillMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		destDir := filepath.Join(basePath, entry.Name())
		meta, err := readMetadata(destDir)
		if err == nil {
			skills = append(skills, meta)
		}
	}

	if jsonOutput {
		data, err := json.MarshalIndent(skills, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tREVISION\tAGENT\tSCOPE\tINSTALL_TIME")
	for _, s := range skills {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Name, s.SourceURL, truncate(s.Revision, 8), s.TargetAgent, s.Scope, s.InstallTime.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
	return nil
}

func truncate(s string, l int) string {
	if len(s) > l {
		return s[:l]
	}
	return s
}
