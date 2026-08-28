//go:build linux || openbsd

package analyze

import (
	"time"

	"golang.org/x/sys/unix"
)

// GetCreationTime returns the creation time (birthtime) of a file on Linux systems.
// Uses statx syscall available on Linux 4.11+.
func GetCreationTime(path string, info os.FileInfo) time.Time {
	var stat unix.Statx_t
	err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &stat)
	if err != nil {
		return time.Time{}
	}
	if stat.Mask&unix.STATX_BTIME == 0 {
		return time.Time{}
	}
	return time.Unix(int64(stat.Btime.Sec), int64(stat.Btime.Nsec))
}
