// Package main - search and serve subcommands for gdu (one binary)
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dundee/gdu/v5/pkg/gdsserve"
	"github.com/dundee/gdu/v5/pkg/indexdir"
	"github.com/dundee/gdu/v5/pkg/indexer"
	"github.com/spf13/cobra"
)

var (
	searchExtFilter     []string
	searchMinSize       int64
	searchMaxSize       int64
	searchSinceDate     string
	searchUntilDate     string
	searchCaseSensitive bool
	searchOutputFormat  string
	searchIndexFile     string
	servePort           int
	serveSocketPath     string
)

func addSearchAndServeCommands(root *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:   "search [-i index] [query]",
		Short: "Search file index",
		Args:  cobra.MaximumNArgs(2),
		RunE:  runSearch,
	}
	searchCmd.Flags().StringVarP(&searchIndexFile, "index", "i", "", "Index file path")
	searchCmd.Flags().StringSliceVar(&searchExtFilter, "ext", nil, "Filter by extensions")
	searchCmd.Flags().Int64Var(&searchMinSize, "min-size", 0, "Min size")
	searchCmd.Flags().Int64Var(&searchMaxSize, "max-size", 0, "Max size")
	searchCmd.Flags().StringVar(&searchSinceDate, "since", "", "Since date")
	searchCmd.Flags().StringVar(&searchUntilDate, "until", "", "Until date")
	searchCmd.Flags().BoolVar(&searchCaseSensitive, "case-sensitive", false, "Case-sensitive")
	searchCmd.Flags().StringVarP(&searchOutputFormat, "format", "f", "simple", "Output: simple, table, json")
	root.AddCommand(searchCmd)

	serveCmd := &cobra.Command{
		Use:   "serve [-i index]",
		Short: "Run search web UI (HTTP + Unix socket)",
		Args:  cobra.NoArgs,
		RunE:  runServe,
	}
	serveCmd.Flags().StringVarP(&searchIndexFile, "index", "i", "", "Index file (auto-detected if omitted)")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8765, "HTTP port")
	serveCmd.Flags().StringVar(&serveSocketPath, "socket", "", "Unix socket path")
	root.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	idxPath := searchIndexFile
	if idxPath == "" {
		// Auto-resolve: try cwd index, then latest
		cwd, _ := os.Getwd()
		if p := indexdir.FindIndexForDir(cwd); p != "" {
			idxPath = p
		} else if p, err := indexdir.FindLatestIndex(); err == nil {
			idxPath = p
		}
	}
	if idxPath == "" {
		return errors.New("no index found. Run gdu on a path first, or use -i <file>")
	}

	svc, err := gdsserve.New(gdsserve.Config{
		IndexPath:  idxPath,
		HTTPPort:   servePort,
		SocketPath: serveSocketPath,
	})
	if err != nil {
		return err
	}
	return svc.Start()
}

func runSearch(cmd *cobra.Command, args []string) error {
	var query, indexPath string
	if searchIndexFile != "" {
		indexPath = searchIndexFile
		if len(args) >= 1 {
			query = args[0]
		}
	} else if len(args) >= 1 {
		if st, err := os.Stat(args[0]); err == nil && !st.IsDir() {
			indexPath = args[0]
			if len(args) >= 2 {
				query = args[1]
			}
		} else {
			query = args[0]
		}
	}
	if indexPath == "" {
		// Try config dir: look for an index matching cwd, then fall back to latest
		cwd, _ := os.Getwd()
		if p := indexdir.FindIndexForDir(cwd); p != "" {
			indexPath = p
		} else if p, err := indexdir.FindLatestIndex(); err == nil {
			indexPath = p
		}
		if indexPath != "" && len(args) >= 1 {
			query = args[0]
		}
	}
	if indexPath == "" {
		return errors.New("no index found. Run gdu on a path first, or use -i <file>")
	}

	sinceTime, _ := parseSearchTime(searchSinceDate)
	untilTime, _ := parseSearchTime(searchUntilDate)

	f, err := os.Open(indexPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := indexer.NewReader(f)
	return searchAndOutput(reader, query, sinceTime, untilTime)
}

func parseSearchTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

func searchAndOutput(reader *indexer.Reader, query string, sinceTime, untilTime time.Time) error {
	q := query
	if !searchCaseSensitive && q != "" {
		q = strings.ToLower(query)
	}
	n := 0
	for {
		entry, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !matchSearchEntry(entry, query, q, sinceTime, untilTime) {
			continue
		}
		outputSearchEntry(entry)
		n++
	}
	if n == 0 {
		fmt.Fprintln(os.Stderr, "No matches")
	} else {
		fmt.Fprintf(os.Stderr, "\n%d matches\n", n)
	}
	return nil
}

func matchSearchEntry(e *indexer.FileEntry, query, qLower string, since, until time.Time) bool {
	if query != "" {
		path := e.Path
		name := e.Name
		if !searchCaseSensitive {
			path = strings.ToLower(path)
			name = strings.ToLower(name)
		}
		if !strings.Contains(path, qLower) && !matchSearchWildcard(name, query) {
			return false
		}
	}
	if len(searchExtFilter) > 0 {
		ok := false
		for _, ext := range searchExtFilter {
			if ext != "" && ext[0] != '.' {
				ext = "." + ext
			}
			if e.Ext == ext {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if searchMinSize > 0 && e.Size < searchMinSize {
		return false
	}
	if searchMaxSize > 0 && e.Size > searchMaxSize {
		return false
	}
	if !since.IsZero() {
		mt, ct := e.GetMtime(), e.GetCtime()
		if (mt.IsZero() || mt.Before(since)) && (ct.IsZero() || ct.Before(since)) {
			return false
		}
	}
	if !until.IsZero() {
		mt, ct := e.GetMtime(), e.GetCtime()
		if !mt.IsZero() && mt.After(until) && !ct.IsZero() && ct.After(until) {
			return false
		}
	}
	return true
}

func matchSearchWildcard(name, pattern string) bool {
	if !searchCaseSensitive {
		name = strings.ToLower(name)
		pattern = strings.ToLower(pattern)
	}
	return matchSearchWildcardRec(name, pattern)
}

func matchSearchWildcardRec(name, pattern string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchSearchWildcardRec(name[i:], pattern) {
					return true
				}
			}
			return false
		case '?':
			if len(name) == 0 {
				return false
			}
			name, pattern = name[1:], pattern[1:]
		default:
			if len(name) == 0 || name[0] != pattern[0] {
				return false
			}
			name, pattern = name[1:], pattern[1:]
		}
	}
	return len(name) == 0
}

func outputSearchEntry(e *indexer.FileEntry) {
	switch searchOutputFormat {
	case "simple":
		fmt.Println(e.Path)
	case "table":
		mt := e.GetMtime()
		mtStr := "-"
		if !mt.IsZero() {
			mtStr = mt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-60s %12s %s\n", truncPath(e.Path, 60), fmtSize(e.Size), mtStr)
	case "json":
		d, _ := e.MarshalNDJSON()
		fmt.Print(string(d))
	default:
		fmt.Println(e.Path)
	}
}

func fmtSize(b int64) string {
	const u = 1024
	if b < u {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(u), 0
	for n := b / u; n >= u; n /= u {
		div *= u
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncPath(p string, n int) string {
	if len(p) <= n {
		return p + strings.Repeat(" ", n-len(p))
	}
	return "..." + p[len(p)-n+3:]
}
