//go:build !darwin && !netbsd && !freebsd && !linux && !openbsd && !windows

package analyze

import (
	"time"
)

// GetCreationTime returns zero time on unsupported platforms
func GetCreationTime(path string, info os.FileInfo) time.Time {
	return time.Time{}
}
