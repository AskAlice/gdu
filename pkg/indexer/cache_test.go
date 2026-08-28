package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	dir := CacheDir()
	if !strings.HasSuffix(dir, filepath.Join("gdu", "indexes")) {
		t.Errorf("CacheDir() = %q, want suffix gdu/indexes", dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("CacheDir() did not create directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("CacheDir() path is not a directory")
	}
}

func TestEncodePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/home/user/projects", "home%user%projects"},
		{"/", "root"},
		{".", "root"},
		{"/tmp", "tmp"},
		{"/a/b/c/d", "a%b%c%d"},
	}
	for _, tt := range tests {
		got := encodePath(tt.in)
		if got != tt.want {
			t.Errorf("encodePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCachePath(t *testing.T) {
	cp := CachePath("/home/user/projects")
	if !strings.HasSuffix(cp, "home%user%projects.ndjson") {
		t.Errorf("CachePath returned %q, want suffix home%%user%%projects.ndjson", cp)
	}
	if !strings.Contains(cp, filepath.Join("gdu", "indexes")) {
		t.Errorf("CachePath returned %q, want to contain gdu/indexes", cp)
	}
}

func TestCacheExistsAndPath(t *testing.T) {
	dir := t.TempDir()
	if CacheExists(dir) {
		t.Fatal("cache should not exist yet")
	}
	cp := CachePath(dir)
	if err := os.WriteFile(cp, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !CacheExists(dir) {
		t.Fatal("cache should exist after creation")
	}
}

func TestFindCacheFile(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}

	cp := CachePath(parent)
	if err := os.WriteFile(cp, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	gotPath, gotDir := FindCacheFile(child)
	if gotPath != cp {
		t.Errorf("FindCacheFile(%q) path = %q, want %q", child, gotPath, cp)
	}
	if gotDir != parent {
		t.Errorf("FindCacheFile(%q) dir = %q, want %q", child, gotDir, parent)
	}
}

func TestFindCacheFileExact(t *testing.T) {
	dir := t.TempDir()
	cp := CachePath(dir)
	if err := os.WriteFile(cp, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	gotPath, gotDir := FindCacheFile(dir)
	if gotPath != cp {
		t.Errorf("FindCacheFile(%q) path = %q, want %q", dir, gotPath, cp)
	}
	if gotDir != dir {
		t.Errorf("FindCacheFile(%q) dir = %q, want %q", dir, gotDir, dir)
	}
}

func TestFindCacheFileNotFound(t *testing.T) {
	dir := t.TempDir()
	got, _ := FindCacheFile(dir)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
