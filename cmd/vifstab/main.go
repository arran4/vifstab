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
				Value: "/root",
				Usage: "Directory to store RCS backups",
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
			cfg := config.Config{
				BackupDir: c.String("backup-dir"),
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
