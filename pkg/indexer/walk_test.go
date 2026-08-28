package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(root, "linkdir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "secret.txt"), filepath.Join(root, "linkfile")); err != nil {
		t.Fatal(err)
	}

	out, err := os.CreateTemp(t.TempDir(), "idx")
	if err != nil {
		t.Fatal(err)
	}
	w := NewWriter(out, 8)
	wk := NewWalker(w, nil, nil, false, false)
	if _, err := wk.Walk(root); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out.Close()

	got := readPaths(t, out.Name())
	if contains(got, filepath.Join(other, "secret.txt")) {
		t.Fatalf("walk followed symlink: %v", got)
	}
	if !contains(got, filepath.Join(root, "real.txt")) {
		t.Fatalf("missing real file: %v", got)
	}
	aliases := 0
	f, err := os.Open(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rd := NewReader(f)
	for rd.Next() {
		e, err := rd.Entry()
		if err == nil && e.Type == "alias" {
			aliases++
		}
	}
	if aliases < 2 {
		t.Fatalf("expected symlink aliases, got %d in %v", aliases, got)
	}
}
