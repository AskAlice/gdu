//go:build !windows && !plan9

package indexer

import (
	"syscall"
)

func deviceID(path string) uint64 {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return 0
	}
	return uint64(stat.Dev)
}
