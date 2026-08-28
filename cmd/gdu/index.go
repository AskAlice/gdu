package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/dundee/gdu/v5/pkg/indexer"
	"github.com/spf13/cobra"
)

var indexFlags struct {
	forceRefresh bool
	showProgress bool
	ignoreDirs   []string
	ignorePats   []string
	noHidden     bool
	noCross      bool
}

var indexCmd = &cobra.Command{
	Use:   "index [path]",
	Short: "Build a file index for fast search with gds",
	Long: `Build a flat NDJSON file index using gdu's parallel scanner.

The index is cached in ~/.cache/gdu/indexes/. Use 'gds' to search it.

Examples:
  gdu index                  # index current directory
  gdu index /home/user       # index /home/user
  gdu index --force-refresh  # rescan ignoring existing cache`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIndex,
}

func init() {
	f := indexCmd.Flags()
	f.BoolVar(&indexFlags.forceRefresh, "force-refresh", false, "Rescan even if cache exists")
	f.BoolVar(&indexFlags.showProgress, "progress", true, "Show progress during scan")
	f.StringSliceVarP(&indexFlags.ignoreDirs, "ignore-dirs", "i", []string{"/proc", "/dev", "/sys", "/run"}, "Paths to ignore")
	f.StringSliceVarP(&indexFlags.ignorePats, "ignore-dirs-pattern", "I", []string{}, "Path patterns to ignore")
	f.BoolVarP(&indexFlags.noHidden, "no-hidden", "H", false, "Ignore hidden directories")
	f.BoolVarP(&indexFlags.noCross, "no-cross", "x", false, "Do not cross filesystem boundaries")
}

func runIndex(cmd *cobra.Command, args []string) error {
	scanPath := "."
	if len(args) > 0 {
		scanPath = args[0]
	}
	abs, err := filepath.Abs(scanPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("stat %s: %w", abs, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", abs)
	}

	cachePath := indexer.CachePath(abs)

	if !indexFlags.forceRefresh && indexer.CacheExists(abs) {
		fmt.Fprintf(os.Stderr, "Cache exists: %s (use --force-refresh to rescan)\n", cachePath)
		return nil
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("creating cache file: %w", err)
	}

	writer := indexer.NewWriter(f, 1024)
	ignoreFn := createIgnoreFunc(indexFlags.ignoreDirs, indexFlags.ignorePats)
	walker := indexer.NewWalker(writer, ignoreFn, nil, indexFlags.noHidden, indexFlags.noCross)

	start := time.Now()

	if indexFlags.showProgress {
		go showProgress(&walker.Stats)
	}

	stats, walkErr := walker.Walk(abs)
	writeErr := writer.Close()
	closeErr := f.Close()

	if indexFlags.showProgress {
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}

	if walkErr != nil {
		return fmt.Errorf("walk error: %w", walkErr)
	}
	if writeErr != nil {
		return fmt.Errorf("write error: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close error: %w", closeErr)
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "Indexed %d files (%s) in %s\n", stats.TotalFiles, formatSize(stats.TotalSize), elapsed.Round(time.Millisecond))
	fmt.Fprintf(os.Stderr, "Cache: %s\n", cachePath)
	return nil
}

func createIgnoreFunc(dirs []string, patterns []string) func(string, string) bool {
	if len(dirs) == 0 && len(patterns) == 0 {
		return nil
	}
	ignoreSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		ignoreSet[d] = true
	}
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return func(name, path string) bool {
		if ignoreSet[name] || ignoreSet[path] {
			return true
		}
		for _, re := range compiled {
			if re.MatchString(name) || re.MatchString(path) {
				return true
			}
		}
		return false
	}
}

func showProgress(stats *indexer.WalkStats) {
	for {
		files := atomic.LoadInt64(&stats.TotalFiles)
		size := atomic.LoadInt64(&stats.TotalSize)
		fmt.Fprintf(os.Stderr, "\r\033[KScanning: %d files (%s)", files, formatSize(size))
		time.Sleep(100 * time.Millisecond)
	}
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

