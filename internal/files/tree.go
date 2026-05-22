package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is a single item in a directory listing.
type Entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Type    string    `json:"type"` // "dir" or "file"
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// Listing is the markdown-filtered contents of one directory.
type Listing struct {
	Path    string  `json:"path"`
	Parent  string  `json:"parent"`
	Entries []Entry `json:"entries"`
}

// Lister produces markdown-only directory listings: every subdirectory is
// shown for navigation, but only markdown files are included.
type Lister struct {
	resolver   *Resolver
	ignore     map[string]bool
	showHidden bool
}

// NewLister builds a Lister. Directory names in ignore are omitted from
// listings; dotfiles and dot-directories are omitted unless showHidden is true.
func NewLister(resolver *Resolver, ignore []string, showHidden bool) *Lister {
	set := make(map[string]bool, len(ignore))
	for _, name := range ignore {
		set[name] = true
	}
	return &Lister{resolver: resolver, ignore: set, showHidden: showHidden}
}

// List returns the directories and markdown files directly inside path,
// sorted with directories first and then alphabetically.
func (l *Lister) List(path string) (Listing, error) {
	abs, err := l.resolver.Resolve(path)
	if err != nil {
		return Listing{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return Listing{}, ErrNotFound
		}
		return Listing{}, err
	}
	if !info.IsDir() {
		return Listing{}, ErrNotDir
	}
	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		return Listing{}, err
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if !l.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if l.ignore[name] {
			continue
		}
		full := filepath.Join(abs, name)

		// A symlink is followed once to decide whether it points at a
		// directory (navigable) or a markdown file.
		isDir := de.IsDir()
		if de.Type()&os.ModeSymlink != 0 {
			if target, err := os.Stat(full); err == nil {
				isDir = target.IsDir()
			} else {
				continue
			}
		}

		if isDir {
			fi, err := de.Info()
			if err != nil {
				continue
			}
			entries = append(entries, Entry{
				Name: name, Path: full, Type: "dir", ModTime: fi.ModTime(),
			})
			continue
		}
		if !IsMarkdown(name) {
			continue
		}
		fi, err := de.Info()
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			Name: name, Path: full, Type: "file", Size: fi.Size(), ModTime: fi.ModTime(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "dir"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := ""
	if abs != l.resolver.Root() {
		parent = filepath.Dir(abs)
	}
	return Listing{Path: abs, Parent: parent, Entries: entries}, nil
}
