# gdu Usage Examples

## Basic Usage

### Analyze a directory (creates index automatically)
```bash
# Analyze current directory
gdu .

# Analyze specific directory
gdu /home/alice/projects

# Analyze with ignored patterns
gdu --ignore-dirs node_modules,.git ~/code
```

**What happens:** gdu scans the directory and automatically creates an index at:
- Linux/macOS: `~/.config/gdu/indexes/home-alice-projects.gds`
- Windows: `%APPDATA%\gdu\indexes\home-alice-projects.gds`

### Exit prompt
When you press Escape or quit gdu, you'll see:
```
Save index and open search? [y/N] 
```

- Type `y` to launch the web UI immediately
- Type `n` to exit and see the command to launch it later:
  ```
  Index saved. To search later, run:
    gdu serve -i "/home/alice/.config/gdu/indexes/home-alice-projects.gds" -p 8765
  ```

## Search Commands

### Search the index (CLI)
```bash
# Auto-detect index from current directory
gdu search *.go

# Case-sensitive search
gdu search --case-sensitive README

# Filter by extension
gdu search --ext go --ext md config

# Filter by size
gdu search --min-size 1048576 large  # files > 1MB

# Filter by date
gdu search --since 2024-01-01 --until 2024-12-31 report

# Different output formats
gdu search *.py --format simple   # paths only (default)
gdu search *.py --format table    # table with size and date
gdu search *.py --format json     # NDJSON output

# Specify index explicitly
gdu search -i ~/.config/gdu/indexes/home-alice.gds "*.log"
```

### Wildcard patterns
```bash
gdu search "*.go"           # all .go files
gdu search "test_*.py"      # files starting with test_
gdu search "config.?"       # config.1, config.2, etc.
gdu search "**/src/*.rs"    # .rs files in any src/ directory
```

## Web UI (Everything-like search)

### Start the web server
```bash
# Auto-detect index
gdu serve

# Specify index and port
gdu serve -i ~/.config/gdu/indexes/home-alice-projects.gds -p 8080

# Custom Unix socket
gdu serve --socket /tmp/gdu-custom.sock
```

**Access:** Open http://localhost:8765 in your browser

**Features:**
- Real-time filtering as you type
- Wildcard support (*.go, test_*)
- Click any result to copy path to clipboard
- Shows file sizes
- Instant results from in-memory index

## Ancestor-Aware Indexing

### Scenario: Nested directories

```bash
# First: Index the parent directory
gdu /home/alice/projects
# Creates: ~/.config/gdu/indexes/home-alice-projects.gds

# Later: Run gdu on a subdirectory
gdu /home/alice/projects/myapp
# Finds existing parent index
# Re-scans /home/alice/projects (NOT just myapp)
# Overwrites: ~/.config/gdu/indexes/home-alice-projects.gds
```

**Why?** This prevents fragmenting into many narrow indexes. One index per directory tree.

### Scenario: Re-running on same path

```bash
gdu ~/documents
# Creates: ~/.config/gdu/indexes/home-alice-documents.gds

# Files added/removed, run again
gdu ~/documents
# Overwrites: ~/.config/gdu/indexes/home-alice-documents.gds
```

**Index is always fresh** - old entries are never kept.

## Integration Examples

### Use with other tools

**Find and open in editor:**
```bash
# Find file and open in vim
vim "$(gdu search main.go --format simple | head -1)"

# Find and copy
gdu search report.pdf --format simple | xargs -I {} cp {} /tmp/
```

**Pipe to fzf:**
```bash
gdu search "" --format simple | fzf
```

**Find largest files:**
```bash
gdu search --min-size 10485760 --format table | sort -k2 -rn
```

## Advanced Examples

### Search with multiple criteria
```bash
# Large Go files modified this year
gdu search --ext go --min-size 51200 --since 2024-01-01 --format table

# Recent logs, case-insensitive
gdu search error --ext log --since 2024-02-01 --format table
```

### Multiple indexes
If you've indexed different directories:
```bash
# Search specific index
gdu search -i ~/.config/gdu/indexes/home-alice.gds config

# Or cd to that directory first (auto-detects)
cd /home/alice
gdu search config
```

### JSON output for scripting
```bash
# Get all Python files as JSON
gdu search --ext py --format json > python_files.ndjson

# Process with jq
gdu search --format json | jq -r 'select(.size > 1000000) | .path'
```

## Directory Structure

Indexes are stored in your config directory:

**Linux/macOS:**
```
~/.config/gdu/indexes/
├── home-alice.gds
├── home-alice-projects.gds
├── home-alice-documents.gds
└── root.gds
```

**Windows:**
```
%APPDATA%\gdu\indexes\
├── C-Users-alice.gds
├── C-Users-alice-Documents.gds
└── D-Projects.gds
```

## Tips

1. **First run on a large directory?** The indexing happens during the normal gdu scan - no extra time!

2. **Want to clean up old indexes?** Just delete from `~/.config/gdu/indexes/` - they're regenerated when needed.

3. **Searching without remembering which index?** Just run `gdu search` in any directory - it auto-detects or uses the most recent index.

4. **Web UI in the background:**
   ```bash
   gdu serve &
   # Now you can search while working
   ```

5. **Quick search shortcut:**
   ```bash
   alias gf='gdu search'
   gf "*.md"
   ```

## Comparison with Everything (Windows)

| Feature | Windows Everything | gdu serve |
|---------|-------------------|-----------|
| Real-time filtering | ✓ | ✓ |
| Wildcard search | ✓ | ✓ |
| Cross-platform | ✗ Windows only | ✓ Linux/Mac/Windows |
| CLI search | ✗ | ✓ |
| Web UI | ✗ | ✓ |
| Storage | System index | `~/.config/gdu/indexes/` |
| Updates | Real-time | On gdu re-run |
