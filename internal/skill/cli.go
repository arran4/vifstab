package skill

import (
	"fmt"
	"os"
)

// RunCLI is the main entrypoint for the `vifstab skill` command group.
func RunCLI(args []string) {
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	subcommand := args[0]
	// subargs := args[1:]

	switch subcommand {
	case "install":
		// vifstab skill install <source> [skill-name] [--agent <agent>] [--scope <scope>]
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: vifstab skill install <source> [skill-name]\n")
			os.Exit(1)
		}
		source := args[1]
		skillName := ""
		if len(args) > 2 && args[2] != "--agent" && args[2] != "--scope" {
			skillName = args[2]
		}

		agent, scope := parseCommonFlags(args)
		err := InstallSkill(source, skillName, agent, scope)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "update":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: vifstab skill update <name> [--force]\n")
			os.Exit(1)
		}
		skillName := args[1]
		force := false
		for _, arg := range args {
			if arg == "--force" {
				force = true
			}
		}
		agent, scope := parseCommonFlags(args)
		err := UpdateSkill(skillName, agent, scope, force)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: vifstab skill remove <name>\n")
			os.Exit(1)
		}
		skillName := args[1]
		agent, scope := parseCommonFlags(args)
		err := RemoveSkill(skillName, agent, scope)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		jsonOutput := false
		for _, arg := range args {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		agent, scope := parseCommonFlags(args)
		err := ListSkills(agent, scope, jsonOutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "inspect":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: vifstab skill inspect <name>\n")
			os.Exit(1)
		}
		skillName := args[1]
		agent, scope := parseCommonFlags(args)
		err := InspectSkill(skillName, agent, scope)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q for %q\n", subcommand, "skill")
		printUsage()
		os.Exit(1)
	}
}

func parseCommonFlags(args []string) (string, string) {
	agent := ""
	scope := ""
	for i, arg := range args {
		if arg == "--agent" && i+1 < len(args) {
			agent = args[i+1]
		}
		if arg == "--scope" && i+1 < len(args) {
			scope = args[i+1]
		}
	}
	return agent, scope
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: vifstab skill <command> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  install   Install a new skill\n")
	fmt.Fprintf(os.Stderr, "  update    Update an installed skill\n")
	fmt.Fprintf(os.Stderr, "  remove    Remove an installed skill\n")
	fmt.Fprintf(os.Stderr, "  list      List installed skills\n")
	fmt.Fprintf(os.Stderr, "  inspect   Inspect an installed skill\n")
}
