package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/editor"
	"github.com/cpuguy83/go-md2man/v2/md2man"
)

//go:embed manpage.md
var manPageMarkdown []byte

func main() {
	var (
		backupDir       string
		file            string
		generateManpage bool
		verifyOnly      bool
	)

	flag.StringVar(&backupDir, "backup-dir", "", "Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)")
	flag.StringVar(&file, "file", "/etc/fstab", "File to edit")
	flag.BoolVar(&generateManpage, "generate-manpage", false, "Generate man page for vifstab to vifstab.1")
	flag.BoolVar(&verifyOnly, "verify", false, "Verify the file without opening an editor")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	if generateManpage {
		manPageTroff := md2man.Render(manPageMarkdown)
		if err := os.WriteFile("vifstab.1", manPageTroff, 0644); err != nil {
			log.Fatal(err)
		}
		return
	}

	if backupDir == "" {
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			backupDir = xdgCache + "/vifstab"
		} else if home, err := os.UserHomeDir(); err == nil {
			backupDir = home + "/.cache/vifstab"
		} else {
			backupDir = "/tmp/.vifstab"
		}
	}

	cfg := config.Config{
		BackupDir: backupDir,
		FilePath:  file,
	}

	if verifyOnly {
		err := editor.VerifyAndLint(cfg)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	err := editor.EditAndLint(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
