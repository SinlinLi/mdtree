// Package files provides safe, root-confined access to markdown files and
// directory listings on the server's filesystem.
package files

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors returned by the files package.
var (
	ErrInvalidPath = errors.New("invalid path")
	ErrOutsideRoot = errors.New("path is outside the configured root")
	ErrNotMarkdown = errors.New("not a markdown file")
	ErrNotFound    = errors.New("not found")
	ErrExists      = errors.New("already exists")
	ErrTooLarge    = errors.New("file is too large")
	ErrIsDir       = errors.New("path is a directory")
	ErrNotDir      = errors.New("path is not a directory")
)

// markdownExts is the set of recognized markdown file extensions.
var markdownExts = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdown":    true,
	".mkd":      true,
	".mkdn":     true,
}

// IsMarkdown reports whether name has a recognized markdown extension.
func IsMarkdown(name string) bool {
	return markdownExts[strings.ToLower(filepath.Ext(name))]
}

// Resolver validates user-supplied paths and confines them to a root directory.
type Resolver struct {
	root           string
	followSymlinks bool
}

// NewResolver returns a Resolver confined to root, which must be an absolute,
// cleaned path. When followSymlinks is false, paths that resolve through a
// symlink to a location outside the root are rejected.
func NewResolver(root string, followSymlinks bool) *Resolver {
	return &Resolver{root: filepath.Clean(root), followSymlinks: followSymlinks}
}

// Root returns the configured root directory.
func (r *Resolver) Root() string { return r.root }

// Resolve cleans and validates p, returning an absolute path guaranteed to be
// within the root. Relative paths are interpreted against the root. The
// returned path may not exist on disk (for create operations).
func (r *Resolver) Resolve(p string) (string, error) {
	if strings.TrimSpace(p) == "" || strings.ContainsRune(p, 0) {
		return "", ErrInvalidPath
	}
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(r.root, abs)
	}
	abs = filepath.Clean(abs)
	if !r.within(abs) {
		return "", ErrOutsideRoot
	}
	if !r.followSymlinks {
		real, err := r.evalSymlinks(abs)
		if err != nil {
			return "", err
		}
		if !r.within(real) {
			return "", ErrOutsideRoot
		}
	}
	return abs, nil
}

// within reports whether p is the root itself or nested inside it.
func (r *Resolver) within(p string) bool {
	if p == r.root {
		return true
	}
	rel, err := filepath.Rel(r.root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// evalSymlinks resolves symlinks in p. For paths that do not yet exist it
// resolves the nearest existing ancestor and re-appends the trailing segments,
// so symlink escapes are caught even when creating new files.
func (r *Resolver) evalSymlinks(p string) (string, error) {
	cur := p
	var trailing []string
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			for i := len(trailing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, trailing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", ErrNotFound
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", ErrNotFound
		}
		trailing = append(trailing, filepath.Base(cur))
		cur = parent
	}
}
