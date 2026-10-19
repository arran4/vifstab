package main

import (
	"io/ioutil"
	"strings"
	"regexp"
)

func main() {
	b, _ := ioutil.ReadFile("cmd/vifstab/main.go")
	s := string(b)

	// Ensure imports
	if !strings.Contains(s, "\"context\"") {
		s = strings.Replace(s, "\"errors\"", "\"context\"\n\t\"errors\"", 1)
	}
	if !strings.Contains(s, "\"github.com/arran4/vifstab/internal/txtar\"") {
		s = strings.Replace(s, "\"github.com/arran4/vifstab/internal/uefi\"", "\"github.com/arran4/vifstab/internal/txtar\"\n\t\"github.com/arran4/vifstab/internal/uefi\"", 1)
	}

	// Re-insert hooks into main()
	hooks := `
	if len(os.Args) > 1 && os.Args[1] == "capture" {
		err := runCapture(os.Args[2:])
		if err != nil {
			handleError(err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "replay" {
		err := runReplay(os.Args[2:])
		if err != nil {
			handleError(err)
		}
		return
	}
`
	if !strings.Contains(s, "os.Args[1] == \"capture\"") {
		s = strings.Replace(s, "if len(os.Args) > 1 && os.Args[1] == \"format\" {", hooks+"\n\tif len(os.Args) > 1 && os.Args[1] == \"format\" {", 1)
	}

	if !strings.Contains(s, "txtarFile string") {
		s = strings.Replace(s, "flag.BoolVar(&verifyOnly, \"verify\", false, \"Verify the file without opening an editor\")", "flag.BoolVar(&verifyOnly, \"verify\", false, \"Verify the file without opening an editor\")\n\tvar txtarFile string\n\tflag.StringVar(&txtarFile, \"txtar\", \"\", \"Run validation against a captured TXTAR snapshot instead of live system\")", 1)
	}

	if !strings.Contains(s, "txtarFile != \"\" && !verifyOnly") {
		s = strings.Replace(s, "if flag.NArg() > 0 {", "\tif txtarFile != \"\" && !verifyOnly {\n\t\thandleError(fmt.Errorf(\"--txtar requires --verify; use 'vifstab replay' for offline validation\"))\n\t}\n\n\tif flag.NArg() > 0 {", 1)
	}

	verify_hook := `if verifyOnly {
		if txtarFile != "" {
			err := runVerifyTxtar(txtarFile)
			if err != nil {
				handleError(err)
			}
			return
		}
		issues, err := editor.VerifyAndLint(cfg, rules.ActualFS{})`

	if !strings.Contains(s, "err := runVerifyTxtar(txtarFile)") {
		s = regexp.MustCompile(`(?s)if verifyOnly \{\n\t\tissues, err := editor\.VerifyAndLint\(cfg, rules\.ActualFS\{\}\)`).ReplaceAllString(s, verify_hook)
	}

	// Add functions at the bottom
	functions := `
func runCapture(args []string) error {
	captureCmd := flag.NewFlagSet("capture", flag.ExitOnError)
	fileFlag := captureCmd.String("file", "/etc/fstab", "File to capture")
	unredactedFlag := captureCmd.Bool("unredacted", false, "Do not redact sensitive values")
	captureCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vifstab capture [options] [<output.txtar>]\n")
		captureCmd.PrintDefaults()
	}
	_ = captureCmd.Parse(args)

	if captureCmd.NArg() > 1 {
		captureCmd.Usage()
		return fmt.Errorf("unexpected arguments: %v", captureCmd.Args()[1:])
	}

	outData, err := txtar.CaptureSystem(context.Background(), rules.ActualFS{}, *fileFlag, *unredactedFlag)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	if captureCmd.NArg() == 1 {
		if err := os.WriteFile(captureCmd.Arg(0), outData, 0600); err != nil {
			return fmt.Errorf("failed to write capture: %w", err)
		}
	} else {
		os.Stdout.Write(outData)
	}
	return nil
}

func runReplay(args []string) error {
	replayCmd := flag.NewFlagSet("replay", flag.ExitOnError)
	replayCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vifstab replay <snapshot.txtar>\n")
		replayCmd.PrintDefaults()
	}
	_ = replayCmd.Parse(args)

	if replayCmd.NArg() != 1 {
		replayCmd.Usage()
		return fmt.Errorf("expected 1 argument, got %d", replayCmd.NArg())
	}

	return runVerifyTxtar(replayCmd.Arg(0))
}

func runVerifyTxtar(txtarPath string) error {
	data, err := os.ReadFile(txtarPath)
	if err != nil {
		return fmt.Errorf("failed to read txtar file: %w", err)
	}

	snapshot, err := txtar.ParseTXTAR(data)
	if err != nil {
		return fmt.Errorf("failed to parse txtar file: %w", err)
	}

	api, err := txtar.ReplaySnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("failed to setup replay environment: %w", err)
	}

	tempFstab, err := os.CreateTemp("", "fstab-replay-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFstab.Name())
	tempFstab.Write(snapshot.Fstab)
	tempFstab.Close()

	cfg, err := config.GetDefaultConfig(tempFstab.Name())
	if err != nil {
		return err
	}

	issues, err := editor.VerifyAndLint(cfg, api)
	if err != nil {
		return err
	}

	var issueCount int
	for _, s := range issues {
		if rules.IsBlockingIssue(s) {
			issueCount++
		}
	}

	if issueCount == 0 {
		fmt.Println("Validation successful.")
		if len(issues) > 0 {
			fmt.Println("Issues:")
			for _, s := range issues {
				fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
			}
		}
		return nil
	}

	fmt.Println("Validation issues found:")
	for _, s := range issues {
		fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
	}

	return &outcome.ApplicationOutcome{
		ExitCode: 1,
		Err:      fmt.Errorf("validation failed with %d blocking issues", issueCount),
	}
}
`
	if !strings.Contains(s, "func runCapture(") {
		s += functions
	}

	ioutil.WriteFile("cmd/vifstab/main.go", []byte(s), 0644)
}
