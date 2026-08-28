package indexer

import (
	"os"
	"path/filepath"
	"strings"
)

// CacheDir returns the centralized cache directory (~/.cache/gdu/indexes/),
// creating it if needed.
func CacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "gdu", "indexes")
	os.MkdirAll(dir, 0o755)
	return dir
}

// encodePath encodes an absolute path for use as a flat filename.
// Leading separator is stripped; remaining separators become %.
// Root "/" becomes "root".
func encodePath(absPath string) string {
	p := filepath.Clean(absPath)
	if p == "/" || p == "." {
		return "root"
	}
	p = strings.TrimLeft(p, string(filepath.Separator))
	return strings.ReplaceAll(p, string(filepath.Separator), "%")
}

// CachePath returns the centralized cache file path for dirPath.
func CachePath(dirPath string) string {
	abs, err := filepath.Abs(dirPath)
	if err != nil {
		abs = dirPath
	}
	return filepath.Join(CacheDir(), encodePath(abs)+".ndjson")
}

// CacheExists checks if a centralized cache file exists for the given directory.
func CacheExists(dirPath string) bool {
	_, err := os.Stat(CachePath(dirPath))
	return err == nil
}

// FindCacheFile checks the centralized cache for startDir, then walks up
// parent directories. Returns the cache file path and the directory it covers.
func FindCacheFile(startDir string) (cachePath string, indexedDir string) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", ""
	}
	for {
		cp := CachePath(dir)
		if _, err := os.Stat(cp); err == nil {
			return cp, dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ""
}
