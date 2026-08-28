//go:build darwin || netbsd || freebsd

package analyze

import (
	"os"
	"syscall"
	"time"
)

// GetCreationTime returns the creation time (birthtime) of a file on BSD/Darwin systems
func GetCreationTime(path string, info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec))
	}
	return time.Time{}
}
