package indexdir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// setXDG overrides $XDG_CONFIG_HOME for the duration of a test.
// t.Setenv automatically restores the original value when the test finishes.
func setXDG(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
}

// --- EncodePath ---

func TestEncodePath_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tests := []struct {
		input string
		want  string
	}{
		{"/home/alice/projects", "home-alice-projects"},
		{"/", "root"},
		{"/usr/local/bin", "usr-local-bin"},
		{"/tmp", "tmp"},
		{"///multiple///slashes///", "multiple-slashes"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EncodePath(tt.input)
			if got != tt.want {
				t.Errorf("EncodePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodePath_Windows(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`C:\Users\alice`, "C-Users-alice"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EncodePath(tt.input)
			if got != tt.want {
				t.Errorf("EncodePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodePath_Empty(t *testing.T) {
	got := EncodePath("")
	if got != "root" {
		t.Errorf("EncodePath(%q) = %q, want %q", "", got, "root")
	}
}

// --- DecodePath ---

func TestDecodePath_Root(t *testing.T) {
	got := DecodePath("root")
	if runtime.GOOS == "windows" {
		if got != `C:\` {
			t.Errorf("DecodePath(root) = %q, want %q", got, `C:\`)
		}
	} else {
		if got != "/" {
			t.Errorf("DecodePath(root) = %q, want %q", got, "/")
		}
	}
}

func TestDecodePath_NonRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	got := DecodePath("home-alice-projects")
	want := "/home/alice/projects"
	if got != want {
		t.Errorf("DecodePath(%q) = %q, want %q", "home-alice-projects", got, want)
	}
}

func TestEncodeDecode_Roundtrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific roundtrip")
	}
	paths := []string{
		"/home/alice",
		"/usr/local/bin",
		"/tmp",
	}
	for _, p := range paths {
		encoded := EncodePath(p)
		decoded := DecodePath(encoded)
		if decoded != p {
			t.Errorf("Roundtrip failed: %q -> %q -> %q", p, encoded, decoded)
		}
	}
}

// --- ConfigDir ---

func TestConfigDir_UsesXDG(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	want := filepath.Join(tmp, "gdu")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
	// Directory should be created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("ConfigDir directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("ConfigDir() path is not a directory")
	}
}

func TestConfigDir_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	dir1, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() first call error: %v", err)
	}
	dir2, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() second call error: %v", err)
	}
	if dir1 != dir2 {
		t.Errorf("ConfigDir() not idempotent: %q != %q", dir1, dir2)
	}
}

// --- IndexDir ---

func TestIndexDir_CreatesSubdir(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	dir, err := IndexDir()
	if err != nil {
		t.Fatalf("IndexDir() error: %v", err)
	}
	want := filepath.Join(tmp, "gdu", "indexes")
	if dir != want {
		t.Errorf("IndexDir() = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("IndexDir directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("IndexDir() path is not a directory")
	}
}

// --- IndexPathForScan ---

func TestIndexPathForScan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	got, err := IndexPathForScan("/home/alice/projects")
	if err != nil {
		t.Fatalf("IndexPathForScan() error: %v", err)
	}
	want := filepath.Join(tmp, "gdu", "indexes", "home-alice-projects.gds")
	if got != want {
		t.Errorf("IndexPathForScan() = %q, want %q", got, want)
	}
}

func TestIndexPathForScan_Root(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	got, err := IndexPathForScan("/")
	if err != nil {
		t.Fatalf("IndexPathForScan() error: %v", err)
	}
	want := filepath.Join(tmp, "gdu", "indexes", "root.gds")
	if got != want {
		t.Errorf("IndexPathForScan(/) = %q, want %q", got, want)
	}
}

// --- ResolveIndexAndScanRoot ---

func TestResolveIndexAndScanRoot_NoExistingIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	// Create a real directory to scan
	scanDir := filepath.Join(tmp, "scanme", "sub")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}

	idxPath, scanRoot, err := ResolveIndexAndScanRoot(scanDir)
	if err != nil {
		t.Fatalf("ResolveIndexAndScanRoot() error: %v", err)
	}

	// Should create a new index for the exact path
	wantIdx := filepath.Join(tmp, "gdu", "indexes", EncodePath(scanDir)+".gds")
	if idxPath != wantIdx {
		t.Errorf("idxPath = %q, want %q", idxPath, wantIdx)
	}
	if scanRoot != scanDir {
		t.Errorf("scanRoot = %q, want %q", scanRoot, scanDir)
	}
}

func TestResolveIndexAndScanRoot_ExactMatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	scanDir := filepath.Join(tmp, "scanme")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create the index file
	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	idxFile := filepath.Join(idxDir, EncodePath(scanDir)+".gds")
	if err := os.WriteFile(idxFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	idxPath, scanRoot, err := ResolveIndexAndScanRoot(scanDir)
	if err != nil {
		t.Fatalf("ResolveIndexAndScanRoot() error: %v", err)
	}
	if idxPath != idxFile {
		t.Errorf("idxPath = %q, want %q", idxPath, idxFile)
	}
	if scanRoot != scanDir {
		t.Errorf("scanRoot = %q, want %q", scanRoot, scanDir)
	}
}

func TestResolveIndexAndScanRoot_AncestorMatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	parentDir := filepath.Join(tmp, "scanme")
	childDir := filepath.Join(parentDir, "sub", "deep")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create an index for the parent
	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentIdx := filepath.Join(idxDir, EncodePath(parentDir)+".gds")
	if err := os.WriteFile(parentIdx, []byte("parent-index"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Request scan of child — should find the parent's index
	idxPath, scanRoot, err := ResolveIndexAndScanRoot(childDir)
	if err != nil {
		t.Fatalf("ResolveIndexAndScanRoot() error: %v", err)
	}
	if idxPath != parentIdx {
		t.Errorf("idxPath = %q, want %q (ancestor)", idxPath, parentIdx)
	}
	if scanRoot != parentDir {
		t.Errorf("scanRoot = %q, want %q (ancestor dir)", scanRoot, parentDir)
	}
}

func TestResolveIndexAndScanRoot_DeepestAncestorWins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	grandparent := filepath.Join(tmp, "a")
	parent := filepath.Join(grandparent, "b")
	child := filepath.Join(parent, "c")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create indexes for both grandparent and parent
	gpIdx := filepath.Join(idxDir, EncodePath(grandparent)+".gds")
	if err := os.WriteFile(gpIdx, []byte("gp"), 0o644); err != nil {
		t.Fatal(err)
	}
	pIdx := filepath.Join(idxDir, EncodePath(parent)+".gds")
	if err := os.WriteFile(pIdx, []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should match the exact path first (child), then walk up to parent (deepest ancestor)
	idxPath, scanRoot, err := ResolveIndexAndScanRoot(child)
	if err != nil {
		t.Fatalf("ResolveIndexAndScanRoot() error: %v", err)
	}
	// The walk starts at child (no index), then checks parent (has index) — returns parent
	if idxPath != pIdx {
		t.Errorf("idxPath = %q, want %q (deepest ancestor)", idxPath, pIdx)
	}
	if scanRoot != parent {
		t.Errorf("scanRoot = %q, want %q", scanRoot, parent)
	}
}

// --- FindIndexForDir ---

func TestFindIndexForDir_Found(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tmp := t.TempDir()
	setXDG(t, tmp)

	absPath := "/home/alice/projects"
	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	idxFile := filepath.Join(idxDir, "home-alice-projects.gds")
	if err := os.WriteFile(idxFile, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := FindIndexForDir(absPath)
	if got != idxFile {
		t.Errorf("FindIndexForDir(%q) = %q, want %q", absPath, got, idxFile)
	}
}

func TestFindIndexForDir_NotFound(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	got := FindIndexForDir("/nonexistent/path")
	if got != "" {
		t.Errorf("FindIndexForDir(nonexistent) = %q, want empty", got)
	}
}

// --- FindLatestIndex ---

func TestFindLatestIndex_Empty(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	// Ensure the indexes dir exists but is empty
	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := FindLatestIndex()
	if err == nil {
		t.Error("FindLatestIndex() should error on empty dir")
	}
}

func TestFindLatestIndex_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(idxDir, "test.gds")
	if err := os.WriteFile(f, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLatestIndex()
	if err != nil {
		t.Fatalf("FindLatestIndex() error: %v", err)
	}
	if got != f {
		t.Errorf("FindLatestIndex() = %q, want %q", got, f)
	}
}

func TestFindLatestIndex_ReturnsNewest(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create older file
	older := filepath.Join(idxDir, "older.gds")
	if err := os.WriteFile(older, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set older modtime to the past
	past := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}

	// Create newer file
	newer := filepath.Join(idxDir, "newer.gds")
	if err := os.WriteFile(newer, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLatestIndex()
	if err != nil {
		t.Fatalf("FindLatestIndex() error: %v", err)
	}
	if got != newer {
		t.Errorf("FindLatestIndex() = %q, want %q (newest)", got, newer)
	}
}

func TestFindLatestIndex_IgnoresNonGDS(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a non-.gds file (should be ignored)
	if err := os.WriteFile(filepath.Join(idxDir, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := FindLatestIndex()
	if err == nil {
		t.Error("FindLatestIndex() should error when only non-.gds files exist")
	}
}

func TestFindLatestIndex_IgnoresDirectories(t *testing.T) {
	tmp := t.TempDir()
	setXDG(t, tmp)

	idxDir := filepath.Join(tmp, "gdu", "indexes")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a directory ending in .gds (should be ignored)
	if err := os.MkdirAll(filepath.Join(idxDir, "fake.gds"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a real .gds file
	real := filepath.Join(idxDir, "real.gds")
	if err := os.WriteFile(real, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLatestIndex()
	if err != nil {
		t.Fatalf("FindLatestIndex() error: %v", err)
	}
	if got != real {
		t.Errorf("FindLatestIndex() = %q, want %q", got, real)
	}
}
