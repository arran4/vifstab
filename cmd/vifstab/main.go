package main

import (
	"log"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/editor"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "vifstab",
		Usage: "Edit fstab files with linting and versioning",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "backup-dir",
				Usage: "Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)",
			},
			&cli.StringFlag{
				Name:  "file",
				Value: "/etc/fstab",
				Usage: "File to edit",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("generate-manpage") {
				man, err := c.App.ToMan()
				if err != nil {
					return err
				}
				return os.WriteFile("vifstab.1", []byte(man), 0644)
			}

			backupDir := c.String("backup-dir")
			if backupDir == "" {
				if c.String("file") == "/etc/fstab" && os.Geteuid() == 0 {
					backupDir = "/root/.vifstab"
				} else {
					// Fallback for non-root or non-/etc/fstab files
					home, err := os.UserHomeDir()
					if err == nil {
						backupDir = home + "/.vifstab"
					} else {
						backupDir = "/tmp/.vifstab"
					}
				}
			}

			cfg := config.Config{
				BackupDir: backupDir,
				FilePath:  c.String("file"),
			}
			return editor.EditAndLint(cfg)
		},
	}

	app.Flags = append(app.Flags, &cli.BoolFlag{
		Name:  "generate-manpage",
		Usage: "Generate man page for vifstab to vifstab.1",
	})

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
