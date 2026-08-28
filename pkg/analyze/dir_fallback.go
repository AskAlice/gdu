//go:build !darwin && !netbsd && !freebsd && !linux && !openbsd && !windows

package analyze

import (
	"os"
)

func getPlatformSpecificUsageAndMli(info os.FileInfo) (usage int64, ino uint64) {
	return info.Size(), 0
}

func setPlatformSpecificAttrs(file *File, f os.FileInfo) {
	file.Mtime = f.ModTime()
	file.Usage = f.Size()
}

func setDirPlatformSpecificAttrs(dir *Dir, path string) {
	stat, err := os.Stat(path)
	if err != nil {
		return
	}
	dir.Mtime = stat.ModTime()
}

func getSyscallStats(info os.FileInfo) (usage int64, mli uint64) {
	return info.Size(), 0
}
