package indexer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dundee/gdu/v5/pkg/analyze"
	"github.com/dundee/gdu/v5/pkg/fs"
)

func TestSaveFromTreeAndRemovePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("yo"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := analyze.CreateAnalyzer().AnalyzeDir(root, func(string, string) bool { return false }, nil)
	dir.UpdateStats(make(fs.HardLinkedItems))
	if err := SaveFromTree(root, dir); err != nil {
		t.Fatal(err)
	}
	cache := CachePath(root)
	got := readPaths(t, cache)
	if !contains(got, filepath.Join(root, "a.txt")) || !contains(got, filepath.Join(sub, "b.txt")) {
		t.Fatalf("missing files: %v", got)
	}

	if err := RemovePath(root, sub); err != nil {
		t.Fatal(err)
	}
	gone := LoadGone(cache)
	if !PathGone(filepath.Join(sub, "b.txt"), gone) {
		t.Fatalf("dir tombstone should hide children: %v", gone)
	}
}

func TestSaveFromTreeSkipsDirSymlink(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := analyze.CreateAnalyzer().AnalyzeDir(root, func(string, string) bool { return false }, nil)
	dir.UpdateStats(make(fs.HardLinkedItems))
	if err := SaveFromTree(root, dir); err != nil {
		t.Fatal(err)
	}
	got := readPaths(t, CachePath(root))
	if contains(got, filepath.Join(other, "secret.txt")) {
		t.Fatalf("followed symlink: %v", got)
	}
	if !contains(got, filepath.Join(root, "real.txt")) {
		t.Fatalf("missing real file: %v", got)
	}
}

func TestDirChanged(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := analyze.CreateAnalyzer().AnalyzeDir(root, func(string, string) bool { return false }, nil)
	dir.UpdateStats(make(fs.HardLinkedItems))
	if DirChanged(dir) {
		t.Fatal("fresh scan should match disk")
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if !DirChanged(dir) {
		t.Fatal("new file should count as a change")
	}
}

func readPaths(t *testing.T, cache string) []string {
	t.Helper()
	f, err := os.Open(cache)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	rd := NewReader(f)
	for rd.Next() {
		e, err := rd.Entry()
		if err != nil || e.Gone {
			continue
		}
		out = append(out, e.Path)
	}
	return out
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
