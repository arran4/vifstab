package vifstab

import _ "embed"

//go:generate go run github.com/arran4/go-subcommand/cmd/gosubc@v0.0.32 generate --dir . --path internal/cli --cli-parser go-flag --timestamp=false --project-provenance=false --emit-go-generate=false --force --prov-version v0.0.32 --prov-commit 4470773c20d143f75eedb0142121dae24e5bcb3a

//go:embed cmd/vifstab/manpage.md
var ManpageMarkdown []byte

// DefaultFile is the default fstab path. Executable tests override this build
// variable to exercise bare invocation without touching the host configuration.
var DefaultFile = "/etc/fstab"
