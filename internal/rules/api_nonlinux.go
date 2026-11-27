//go:build !linux

package rules

func extractLinuxACL(path string, res *FileInfoResult) {}
