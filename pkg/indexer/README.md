# pkg/indexer - DEPENDENCY REQUIRED

This package is required for the config directory index feature but was created in previous work.

## Required Files

From previous implementation sessions, `pkg/indexer` should contain:

### Core Files
- `format.go` - Binary format definition (magic bytes, version, record structure)
- `write.go` - `Writer` type with `Write(*FileEntry)` and `Close()` methods
- `read.go` - `Reader` type with `Next() (*FileEntry, error)` method
- `entry.go` - `FileEntry` type with fields and marshaling

### FileEntry Structure
```go
type FileEntry struct {
    Name  string
    Path  string
    Size  int64
    Mtime int64  // Unix seconds
    Ctime int64  // Unix seconds
    Ext   string
}

func (e *FileEntry) GetMtime() time.Time
func (e *FileEntry) GetCtime() time.Time
func (e *FileEntry) MarshalNDJSON() ([]byte, error)
```

### Binary Format
- Magic: `gds\x01`
- Version: 1 byte
- Records: path_len (uint16) + path + size (int64) + mtime_ns (int64) + ctime_ns (int64) + ext_len (uint8) + ext

### Tests
- `format_test.go` - Format encoding/decoding tests
- `integration_test.go` - Full read/write cycle tests

## Integration Status

**This package exists in previous work** and needs to be integrated from:
- Previous conversation transcripts
- Or re-implemented following the specification above
- Or imported from the branch where it was originally created

## Current Usage

This feature branch requires `pkg/indexer` for:
- `cmd/gdu/subcommands.go` - Uses `Reader` and `FileEntry` for search
- `pkg/gdsserve/server.go` - Uses `Reader` and `FileEntry` for web UI

## Quick Fix

To build this branch, you need to either:
1. Merge the pkg/indexer implementation from previous work
2. Or temporarily stub it out for testing the config directory logic

See `/Users/alice/.cursor/projects/Users-alice-codex-worktrees-8ac7-gdu/agent-transcripts/69c7ae50-839a-4816-bb90-ebfb5f06aa4e.txt` 
for the full conversation history where pkg/indexer was originally created.
