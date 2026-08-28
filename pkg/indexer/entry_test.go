package indexer

import (
	"testing"
	"time"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	e := FileEntry{
		Name: "hello.go", Path: "/tmp/hello.go",
		Size: 1234, Mtime: 1700000000, Ctime: 1699999000, Ext: ".go",
	}
	b, err := e.MarshalNDJSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalNDJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != e {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, e)
	}
}

func TestGetMtimeCtime(t *testing.T) {
	e := FileEntry{Mtime: 1700000000, Ctime: 0}
	if e.GetMtime().IsZero() {
		t.Fatal("mtime should not be zero")
	}
	if !e.GetCtime().IsZero() {
		t.Fatal("ctime should be zero when Ctime == 0")
	}
	e.Ctime = 1699999000
	ct := e.GetCtime()
	if ct.Equal(time.Time{}) {
		t.Fatal("ctime should not be zero")
	}
}

func TestFileEntryNoExt(t *testing.T) {
	e := FileEntry{Name: "Makefile", Path: "/tmp/Makefile", Size: 100, Mtime: 1700000000}
	b, err := e.MarshalNDJSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalNDJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ext != "" {
		t.Fatalf("expected empty ext, got %q", got.Ext)
	}
}
