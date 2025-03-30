//go:build !windows

package main

import (
	"os"
	"syscall"
)

func fileStatOf(info os.FileInfo) (int, int, int64) {
	mode := int64(0644)
	if info.IsDir() {
		mode = int64(0755)
	}

	uid, gid := 0, 0
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		uid, gid, mode = int(st.Uid), int(st.Gid), int64(st.Mode)
	}

	return uid, gid, mode
}
