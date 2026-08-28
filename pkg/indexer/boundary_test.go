package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsUnder(t *testing.T) {
	if !isUnder("/home/a/b", "/home/a") {
		t.Fatal("child should be under parent")
	}
	if isUnder("/home/ab", "/home/a") {
		t.Fatal("sibling prefix must not count")
	}
	if !isUnder("/home/a", "/home/a") {
		t.Fatal("path is under itself")
	}
}

func TestIsCacheName(t *testing.T) {
	if !IsCacheName(".gdu-cache-root.ndjson") {
		t.Fatal("expected cache name")
	}
	if IsCacheName("readme.md") {
		t.Fatal("not a cache name")
	}
}

func TestDecideSymlinkNeverDescends(t *testing.T) {
	dir := t.TempDir()
	b := newBoundary(dir)
	ok, tgt := b.decide(dir, filepath.Join(dir, "link"), true, "/elsewhere")
	if ok {
		t.Fatal("symlink must not descend")
	}
	if tgt != "/elsewhere" {
		t.Fatalf("target=%q", tgt)
	}
}

func TestPathGone(t *testing.T) {
	sep := string(os.PathSeparator)
	gone := []string{"/tmp/del"}
	if !PathGone("/tmp/del", gone) || !PathGone("/tmp/del"+sep+"x", gone) {
		t.Fatal("prefix should match")
	}
	if PathGone("/tmp/delta", gone) {
		t.Fatal("must not match sibling prefix")
	}
}
