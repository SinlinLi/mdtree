// Package search maintains an in-memory index of markdown file paths and
// answers fuzzy filename queries against it.
package search

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SinlinLi/mdtree/internal/files"
)

// doc is one indexed markdown file with precomputed lowercase fields.
type doc struct {
	path      string
	name      string
	lowerPath string
	lowerName string
}

func newDoc(path string) doc {
	name := filepath.Base(path)
	return doc{
		path:      path,
		name:      name,
		lowerPath: strings.ToLower(path),
		lowerName: strings.ToLower(name),
	}
}

// Index is a concurrency-safe in-memory index of markdown files. It is
// populated by Build and kept current by Add, Remove and Rename.
type Index struct {
	root     string
	ignore   map[string]bool
	maxFiles int
	log      *slog.Logger

	mu        sync.RWMutex
	docs      map[string]doc // keyed by absolute path
	builtAt   time.Time
	lastBuild time.Duration
}

// NewIndex creates an empty index rooted at root. Directory names in ignore
// are skipped during Build; maxFiles caps the number of indexed files.
func NewIndex(root string, ignore []string, maxFiles int, log *slog.Logger) *Index {
	set := make(map[string]bool, len(ignore))
	for _, name := range ignore {
		set[name] = true
	}
	if maxFiles <= 0 {
		maxFiles = 200000
	}
	return &Index{
		root:     root,
		ignore:   set,
		maxFiles: maxFiles,
		log:      log,
		docs:     make(map[string]doc),
	}
}

// Build walks the root directory tree and (re)populates the index. Unreadable
// directories are skipped rather than aborting the walk. The freshly built doc
// set is swapped in atomically, so concurrent queries are never disrupted.
func (idx *Index) Build() (int, time.Duration, error) {
	start := time.Now()
	docs := make(map[string]doc)
	capped := false

	walkErr := filepath.WalkDir(idx.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir // unreadable directory: skip its subtree
			}
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != idx.root && (idx.ignore[name] || strings.HasPrefix(name, ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if len(docs) >= idx.maxFiles {
			capped = true
			return filepath.SkipAll
		}
		if files.IsMarkdown(d.Name()) {
			docs[path] = newDoc(path)
		}
		return nil
	})
	if walkErr != nil {
		return 0, time.Since(start), walkErr
	}

	elapsed := time.Since(start)
	idx.mu.Lock()
	idx.docs = docs
	idx.builtAt = time.Now()
	idx.lastBuild = elapsed
	idx.mu.Unlock()

	if idx.log != nil {
		idx.log.Info("search index built",
			slog.Int("files", len(docs)),
			slog.Duration("elapsed", elapsed),
			slog.Bool("capped", capped))
	}
	return len(docs), elapsed, nil
}

// Add inserts or updates a single markdown file in the index.
func (idx *Index) Add(path string) {
	if !files.IsMarkdown(path) {
		return
	}
	idx.mu.Lock()
	idx.docs[path] = newDoc(path)
	idx.mu.Unlock()
}

// Remove deletes a file from the index.
func (idx *Index) Remove(path string) {
	idx.mu.Lock()
	delete(idx.docs, path)
	idx.mu.Unlock()
}

// Rename updates the index after a file move.
func (idx *Index) Rename(from, to string) {
	idx.mu.Lock()
	delete(idx.docs, from)
	if files.IsMarkdown(to) {
		idx.docs[to] = newDoc(to)
	}
	idx.mu.Unlock()
}

// Stats describes the current index state.
type Stats struct {
	Files       int       `json:"files"`
	BuiltAt     time.Time `json:"builtAt"`
	BuildMillis float64   `json:"buildMillis"`
}

// Stats returns a snapshot of index statistics.
func (idx *Index) Stats() Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return Stats{
		Files:       len(idx.docs),
		BuiltAt:     idx.builtAt,
		BuildMillis: float64(idx.lastBuild.Microseconds()) / 1000.0,
	}
}
