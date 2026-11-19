//go:build unix

package rules

import (
	"io/fs"
	"syscall"
)

func extractOSFileInfo(info fs.FileInfo, res *FileInfoResult) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid := int(stat.Uid)
		gid := int(stat.Gid)
		res.UID = &uid
		res.GID = &gid
	}
}
