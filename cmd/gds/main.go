package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dundee/gdu/v5/pkg/indexer"
	"github.com/spf13/cobra"
)

var (
	extFilter     []string
	minSize       int64
	maxSize       int64
	sinceDate     string
	untilDate     string
	caseSensitive bool
	outputFormat  string
	searchDir     string
)

var rootCmd = &cobra.Command{
	Use:   "gds [query]",
	Short: "Search a gdu file index",
	Long: `Search an NDJSON file index created by 'gdu index'.

Finds the cached index in ~/.cache/gdu/indexes/.

Examples:
  gds "*.go"                    # find Go files
  gds --ext .rs,.go "parser"    # search with extension filter
  gds --min-size 1048576        # files >= 1 MiB
  gds -d /home/user "readme"    # search from specific dir`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func init() {
	f := rootCmd.Flags()
	f.StringSliceVar(&extFilter, "ext", nil, "Filter by extension (e.g., --ext .go,.rs)")
	f.Int64Var(&minSize, "min-size", 0, "Minimum file size in bytes")
	f.Int64Var(&maxSize, "max-size", 0, "Maximum file size in bytes (0 = no limit)")
	f.StringVar(&sinceDate, "since", "", "Files modified since date (YYYY-MM-DD or RFC3339)")
	f.StringVar(&untilDate, "until", "", "Files modified until date (YYYY-MM-DD or RFC3339)")
	f.BoolVar(&caseSensitive, "case-sensitive", false, "Case-sensitive query matching")
	f.StringVar(&outputFormat, "format", "simple", "Output format: simple, json")
	f.StringVarP(&searchDir, "dir", "d", "", "Start directory for finding cache (default: cwd)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	startDir := searchDir
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting cwd: %w", err)
		}
	}

	cachePath, _ := indexer.FindCacheFile(startDir)
	if cachePath == "" {
		return fmt.Errorf("no index found (run 'gdu index' first)")
	}

	f, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("opening index: %w", err)
	}
	defer f.Close()

	var sinceT, untilT time.Time
	if sinceDate != "" {
		sinceT, err = parseTime(sinceDate)
		if err != nil {
			return fmt.Errorf("parsing --since: %w", err)
		}
	}
	if untilDate != "" {
		untilT, err = parseTime(untilDate)
		if err != nil {
			return fmt.Errorf("parsing --until: %w", err)
		}
	}

	gone := indexer.LoadGone(cachePath)
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("rewinding index: %w", err)
	}
	reader := indexer.NewReader(f)
	matched := 0
	for reader.Next() {
		entry, err := reader.Entry()
		if err != nil {
			continue
		}
		if entry.Gone || indexer.PathGone(entry.Path, gone) {
			continue
		}
		if !matchEntry(entry, query, sinceT, untilT) {
			continue
		}
		outputEntry(entry)
		matched++
	}
	if err := reader.Err(); err != nil {
		return fmt.Errorf("reading index: %w", err)
	}

	if matched == 0 && query != "" {
		fmt.Fprintf(os.Stderr, "No matches for %q\n", query)
	}
	return nil
}

func matchEntry(e indexer.FileEntry, query string, since, until time.Time) bool {
	if len(extFilter) > 0 {
		ext := strings.ToLower(e.Ext)
		found := false
		for _, ef := range extFilter {
			if ext == strings.ToLower(ef) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if minSize > 0 && e.Size < minSize {
		return false
	}
	if maxSize > 0 && e.Size > maxSize {
		return false
	}
	if !since.IsZero() && e.GetMtime().Before(since) {
		return false
	}
	if !until.IsZero() && e.GetMtime().After(until) {
		return false
	}
	if query == "" {
		return true
	}

	name := e.Name
	path := e.Path
	q := query
	if !caseSensitive {
		name = strings.ToLower(name)
		path = strings.ToLower(path)
		q = strings.ToLower(q)
	}

	if strings.ContainsAny(q, "*?") {
		return matchWild(q, name) || matchWild(q, path)
	}
	return strings.Contains(name, q) || strings.Contains(path, q)
}

func matchWild(pattern, s string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchWild(pattern, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		default:
			if len(s) == 0 || pattern[0] != s[0] {
				return false
			}
			pattern = pattern[1:]
			s = s[1:]
		}
	}
	return len(s) == 0
}

func outputEntry(e indexer.FileEntry) {
	switch outputFormat {
	case "json":
		b, _ := json.Marshal(e)
		fmt.Println(string(b))
	default:
		fmt.Println(e.Path)
	}
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD or RFC3339)", s)
}
