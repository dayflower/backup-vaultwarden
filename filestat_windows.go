//go:build windows

package main

import (
	"os"
)

func fileStatOf(info os.FileInfo) (int, int, int64) {
	mode := int64(0644)
	if info.IsDir() {
		mode = int64(0755)
	}

	uid, gid := 0, 0

	return uid, gid, mode
}
