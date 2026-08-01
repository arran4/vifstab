package rcs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/golang-rcs"
)

type Config struct {
	Dir string
}

func Update(filePath string, content []byte, config Config) error {
	if config.Dir == "" {
		return nil
	}

	if err := os.MkdirAll(config.Dir, 0700); err != nil {
		return fmt.Errorf("failed to create rcs dir: %w", err)
	}

	baseName := filepath.Base(filePath)
	rcsFilepath := filepath.Join(config.Dir, baseName+",v")

	var f *rcs.File

	if _, err := os.Stat(rcsFilepath); os.IsNotExist(err) {
		f = rcs.NewFile()
		f.Head = "1.1"
		f.Access = true
		f.Strict = true

		f.RevisionHeads = append(f.RevisionHeads, &rcs.RevisionHead{
			Revision: "1.1",
			Date:     rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
			Author:   rcs.ID(os.Getenv("USER")),
			State:    "Exp",
		})

		f.RevisionContents = append(f.RevisionContents, &rcs.RevisionContent{
			Revision: "1.1",
			Log:      "Initial revision",
			Text:     string(content),
		})
	} else {
		rcsData, err := os.Open(rcsFilepath)
		if err != nil {
			return fmt.Errorf("failed to open rcs file: %w", err)
		}
		defer func() {
			_ = rcsData.Close()
		}()

		f, err = rcs.ParseFile(rcsData)
		if err != nil {
			return fmt.Errorf("failed to parse rcs file: %w", err)
		}

		headParts := strings.Split(f.Head, ".")
		if len(headParts) == 2 {
			minor := headParts[1]
			newMinor, err := strconv.Atoi(minor)
			if err != nil {
				return fmt.Errorf("failed to parse minor revision %q: %w", minor, err)
			}
			newHead := fmt.Sprintf("%s.%d", headParts[0], newMinor+1)
			f.Head = newHead

			f.RevisionHeads = append([]*rcs.RevisionHead{{
				Revision: rcs.Num(newHead),
				Date:     rcs.DateTime(time.Now().UTC().Format(rcs.DateFormat)),
				Author:   rcs.ID(os.Getenv("USER")),
				State:    "Exp",
			}}, f.RevisionHeads...)

			f.RevisionContents = append([]*rcs.RevisionContent{{
				Revision: newHead,
				Log:      "Updated revision",
				Text:     string(content),
			}}, f.RevisionContents...)
		}
	}

	err := os.WriteFile(rcsFilepath, []byte(f.String()), 0600)
	if err != nil {
		return fmt.Errorf("failed to write rcs file: %w", err)
	}

	return nil
}
