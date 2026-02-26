package analyze

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dundee/gdu/v5/pkg/fs"
	"github.com/dundee/gdu/v5/pkg/indexer"
)

// LoadDirFromIndex reads an index file and builds an in-memory Dir tree for the given scan root.
// The root Dir has Name = filepath.Base(scanRoot) and BasePath = filepath.Dir(scanRoot).
// Returns nil if the file does not exist or cannot be read.
func LoadDirFromIndex(indexPath, scanRoot string) (*Dir, error) {
	f, err := os.Open(indexPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanRoot = filepath.Clean(scanRoot)
	if !filepath.IsAbs(scanRoot) {
		abs, err := filepath.Abs(scanRoot)
		if err != nil {
			return nil, err
		}
		scanRoot = abs
	}

	rootName := filepath.Base(scanRoot)
	basePath := filepath.Dir(scanRoot)
	root := &Dir{
		File: &File{
			Name:  rootName,
			Flag:  ' ',
			Size:  0,
			Usage: 0,
		},
		BasePath:  basePath,
		Files:     make(fs.Files, 0),
		ItemCount: 1,
	}

	// dirsByPath maps normalized relative path (e.g. "foo" or "foo/bar") -> *Dir
	dirsByPath := map[string]*Dir{"": root}

	reader := indexer.NewReader(f)
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		entryPath := filepath.Clean(entry.Path)
		if !filepath.IsAbs(entryPath) {
			entryPath = filepath.Join(scanRoot, entryPath)
		}
		rel, err := filepath.Rel(scanRoot, entryPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		mtime := entry.GetMtime()
		if mtime.IsZero() {
			mtime = time.Now()
		}

		if entry.Size == 0 && entry.Name != "" {
			// Directory (we write dirs with size 0)
			dirPath := rel
			parentPath := filepath.Dir(dirPath)
			if parentPath == "." {
				parentPath = ""
			}
			parent, ok := dirsByPath[parentPath]
			if !ok {
				continue
			}
			name := filepath.Base(dirPath)
			if name == "." || name == "" {
				continue
			}
			if existing := findChildByName(parent, name); existing != nil {
				continue
			}
			d := &Dir{
				File: &File{
					Name:  name,
					Flag:  ' ',
					Size:  0,
					Usage: 0,
					Mtime: mtime,
					Parent: parent,
				},
				BasePath:  "",
				Files:     make(fs.Files, 0),
				ItemCount: 1,
			}
			parent.AddFile(d)
			dirsByPath[dirPath] = d
			continue
		}

		// File
		parentPath := filepath.Dir(rel)
		if parentPath == "." {
			parentPath = ""
		}
		parent, ok := dirsByPath[parentPath]
		if !ok {
			// Ensure parent chain exists
			segments := strings.Split(filepath.ToSlash(parentPath), "/")
			var curPath string
			parent = root
			for _, seg := range segments {
				if seg == "" {
					continue
				}
				if curPath != "" {
					curPath += string(filepath.Separator) + seg
				} else {
					curPath = seg
				}
				if d, exists := dirsByPath[curPath]; exists {
					parent = d
					continue
				}
				d := &Dir{
					File: &File{
						Name:   seg,
						Flag:   ' ',
						Size:   0,
						Usage:  0,
						Mtime:  mtime,
						Parent: parent,
					},
					BasePath:  "",
					Files:     make(fs.Files, 0),
					ItemCount: 1,
				}
				parent.AddFile(d)
				dirsByPath[curPath] = d
				parent = d
			}
		}

		if findChildByName(parent, entry.Name) != nil {
			continue
		}
		file := &File{
			Name:   entry.Name,
			Flag:   ' ',
			Size:   entry.Size,
			Usage:  entry.Size,
			Mtime:  mtime,
			Parent: parent,
		}
		parent.AddFile(file)
	}

	root.UpdateStats(make(fs.HardLinkedItems))
	return root, nil
}

func findChildByName(parent *Dir, name string) fs.Item {
	for _, item := range parent.Files {
		if item.GetName() == name {
			return item
		}
	}
	return nil
}
