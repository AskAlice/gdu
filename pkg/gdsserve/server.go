package gdsserve

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dundee/gdu/v5/pkg/indexer"
)

// Server serves the index over HTTP and Unix socket with an Everything-like search UI
type Server struct {
	entries   []*indexer.FileEntry
	indexPath  string
	httpPort   int
	socketPath string
	mu         sync.RWMutex
}

// Config holds server configuration
type Config struct {
	IndexPath  string
	HTTPPort   int
	SocketPath string
}

// New creates a new Server that loads the index for fast search
func New(cfg Config) (*Server, error) {
	f, err := os.Open(cfg.IndexPath)
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer f.Close()

	var entries []*indexer.FileEntry
	reader := indexer.NewReader(f)
	for {
		e, err := reader.Next()
		if err != nil {
			break
		}
		entries = append(entries, e)
	}

	if cfg.SocketPath == "" {
		cfg.SocketPath = filepath.Join(os.TempDir(), "gdu.sock")
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 8765
	}

	return &Server{
		entries:   entries,
		indexPath: cfg.IndexPath,
		httpPort:  cfg.HTTPPort,
		socketPath: cfg.SocketPath,
	}, nil
}

// Search filters entries by query (substring match on path, case-insensitive by default)
func (s *Server) Search(query string, caseSensitive bool, maxResults int) []*indexer.FileEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if maxResults <= 0 {
		maxResults = 500
	}
	q := query
	if !caseSensitive {
		q = strings.ToLower(query)
	}

	var out []*indexer.FileEntry
	for _, e := range s.entries {
		if len(out) >= maxResults {
			break
		}
		path := e.Path
		if !caseSensitive {
			path = strings.ToLower(path)
		}
		if query == "" || strings.Contains(path, q) || matchWildcard(e.Name, query, caseSensitive) {
			out = append(out, e)
		}
	}
	return out
}

func matchWildcard(name, pattern string, caseSensitive bool) bool {
	if !caseSensitive {
		name = strings.ToLower(name)
		pattern = strings.ToLower(pattern)
	}
	return matchWildcardRecursive(name, pattern)
}

func matchWildcardRecursive(name, pattern string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchWildcardRecursive(name[i:], pattern) {
					return true
				}
			}
			return false
		case '?':
			if len(name) == 0 {
				return false
			}
			name = name[1:]
			pattern = pattern[1:]
		default:
			if len(name) == 0 || name[0] != pattern[0] {
				return false
			}
			name = name[1:]
			pattern = pattern[1:]
		}
	}
	return len(name) == 0
}

// Start runs both HTTP and Unix socket servers
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveUI)
	mux.HandleFunc("/search", s.serveSearch)
	mux.HandleFunc("/api/search", s.serveSearchAPI)

	// Unix socket
	os.Remove(s.socketPath)
	unixListener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("unix socket: %w", err)
	}
	go func() {
		_ = http.Serve(unixListener, mux)
	}()

	// HTTP
	addr := fmt.Sprintf(":%d", s.httpPort)
	fmt.Fprintf(os.Stderr, "gdu serve: http://localhost%s  unix:%s\n", addr, s.socketPath)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, uiHTML)
}

func (s *Server) serveSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	caseSensitive := r.URL.Query().Get("case") == "1"
	results := s.Search(q, caseSensitive, 200)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   q,
		"results": results,
		"count":   len(results),
	})
}

func (s *Server) serveSearchAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" && r.Method == http.MethodPost {
		r.ParseForm()
		q = r.FormValue("q")
	}
	q = strings.TrimSpace(q)
	caseSensitive := r.URL.Query().Get("case") == "1"
	results := s.Search(q, caseSensitive, 500)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   q,
		"results": results,
		"count":   len(results),
	})
}
