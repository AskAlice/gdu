// Package indexdir provides cross-platform index storage in the user's config directory.
// Indexes are saved as <encoded-path>.gds files inside ~/.config/gdu/indexes/ (Linux/macOS)
// or %APPDATA%\gdu\indexes\ (Windows).
package indexdir

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ConfigDir returns the gdu config directory, creating it if necessary.
// Uses $XDG_CONFIG_HOME/gdu if set, otherwise falls back to os.UserConfigDir()/gdu.
// On macOS this gives ~/.config/gdu (via XDG) rather than ~/Library/Application Support/gdu.
func ConfigDir() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		base = xdg
	} else {
		d, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		base = d
	}
	dir := filepath.Join(base, "gdu")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// IndexDir returns the directory where index files are stored, creating it if necessary.
func IndexDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, "indexes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// EncodePath converts an absolute filesystem path to a safe filename component.
// Separators (/, \) and colons are replaced with dashes, leading dashes are trimmed,
// and consecutive dashes are collapsed.
//
//	/home/alice/projects -> home-alice-projects
//	/                    -> root
//	C:\Users\alice       -> C-Users-alice
func EncodePath(absPath string) string {
	// Replace all path separators and colons with dashes
	p := absPath
	p = strings.ReplaceAll(p, "\\", "-")
	p = strings.ReplaceAll(p, "/", "-")
	p = strings.ReplaceAll(p, ":", "-")

	// Collapse consecutive dashes and trim
	for strings.Contains(p, "--") {
		p = strings.ReplaceAll(p, "--", "-")
	}
	p = strings.Trim(p, "-")

	if p == "" {
		return "root"
	}
	return p
}

// DecodePath is a best-effort reverse of EncodePath. On non-Windows it prepends "/".
func DecodePath(encoded string) string {
	if encoded == "root" {
		if runtime.GOOS == "windows" {
			return `C:\`
		}
		return "/"
	}
	p := strings.ReplaceAll(encoded, "-", string(filepath.Separator))
	if runtime.GOOS != "windows" {
		p = string(filepath.Separator) + p
	}
	return p
}

// IndexPathForScan returns the deterministic index file path for a given absolute scan path.
// This does NOT check for ancestor indexes — use ResolveIndexAndScanRoot for that.
func IndexPathForScan(absPath string) (string, error) {
	dir, err := IndexDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, EncodePath(absPath)+".gds"), nil
}

// ResolveIndexAndScanRoot determines which index file to use and what directory to scan.
//
// It walks up from requestedPath checking for existing ancestor indexes:
//   - If an exact match or ancestor index exists, returns that index path and the
//     ancestor's absolute path as the scan root (so the full index is refreshed).
//   - If no ancestor index exists, returns a new index path for requestedPath.
//
// This prevents creating redundant narrow indexes when a broader one already exists.
func ResolveIndexAndScanRoot(requestedPath string) (indexPath string, scanRoot string, err error) {
	dir, err := IndexDir()
	if err != nil {
		return "", "", err
	}

	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", "", err
	}

	// Walk up from the requested path to the filesystem root,
	// checking if an index exists for each ancestor.
	candidate := absPath
	for {
		encoded := EncodePath(candidate)
		idxFile := filepath.Join(dir, encoded+".gds")
		if _, serr := os.Stat(idxFile); serr == nil {
			return idxFile, candidate, nil
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			// Reached filesystem root, no ancestor index found
			break
		}
		candidate = parent
	}

	// No ancestor index found — create a new index for the requested path
	idxPath := filepath.Join(dir, EncodePath(absPath)+".gds")
	return idxPath, absPath, nil
}

// FindIndexForDir returns the index file for the given absolute path, or "" if none exists.
func FindIndexForDir(absPath string) string {
	dir, err := IndexDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, EncodePath(absPath)+".gds")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// FindLatestIndex returns the most recently modified .gds file in the index directory.
// Useful as a fallback when no specific index can be determined.
func FindLatestIndex() (string, error) {
	dir, err := IndexDir()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var best string
	var bestTime int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".gds") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); t > bestTime {
			bestTime = t
			best = filepath.Join(dir, e.Name())
		}
	}
	if best == "" {
		return "", os.ErrNotExist
	}
	return best, nil
}
