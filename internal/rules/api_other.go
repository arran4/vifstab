//go:build !unix

package rules

import "io/fs"

func extractOSFileInfo(info fs.FileInfo, res *FileInfoResult) {}
