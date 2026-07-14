package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/editor"
)

const manPageContent = `.nh
.TH vifstab 8

.SH NAME
vifstab \- Edit fstab files with linting and versioning


.SH SYNOPSIS
vifstab

.EX
[--backup-dir]=[value]
[--file]=[value]
[--generate-manpage]
[--help|-h]
.EE

.PP
\fBUsage\fP:

.EX
vifstab [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
.EE


.SH GLOBAL OPTIONS
\fB--backup-dir\fP="": Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)

.PP
\fB--file\fP="": File to edit (default: "/etc/fstab")

.PP
\fB--generate-manpage\fP: Generate man page for vifstab to vifstab.1

.PP
\fB--help, -h\fP: show help


.SH COMMANDS
.SH help, h
Shows a list of commands or help for one command
`

func main() {
	var (
		backupDir       string
		file            string
		generateManpage bool
	)

	flag.StringVar(&backupDir, "backup-dir", "", "Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)")
	flag.StringVar(&file, "file", "/etc/fstab", "File to edit")
	flag.BoolVar(&generateManpage, "generate-manpage", false, "Generate man page for vifstab to vifstab.1")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if generateManpage {
		if err := os.WriteFile("vifstab.1", []byte(manPageContent), 0644); err != nil {
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

	err := editor.EditAndLint(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
