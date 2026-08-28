//go:build windows

package analyze

import (
	"syscall"
	"time"
)

// GetCreationTime returns the creation time of a file on Windows systems
func GetCreationTime(path string, info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, stat.CreationTime.Nanoseconds())
	}
	return time.Time{}
}
