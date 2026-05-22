package files

import (
	"os"
	"path/filepath"
	"time"
)

// MaxFileSize is the largest markdown file mdtree will read or write.
const MaxFileSize = 10 << 20 // 10 MiB

// FileInfo describes a markdown file without its content.
type FileInfo struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// FileContent is a markdown file together with its UTF-8 text content.
type FileContent struct {
	FileInfo
	Content string `json:"content"`
}

// Manager performs read and write operations on markdown files within the
// configured root.
type Manager struct {
	resolver *Resolver
}

// NewManager builds a Manager bound to resolver.
func NewManager(resolver *Resolver) *Manager {
	return &Manager{resolver: resolver}
}

// ResolvePath validates p and returns its canonical absolute form. It is used
// by callers (such as the API layer) that need the canonical path for index
// bookkeeping.
func (m *Manager) ResolvePath(p string) (string, error) {
	return m.resolver.Resolve(p)
}

func fileInfoOf(abs string, fi os.FileInfo) FileInfo {
	return FileInfo{
		Path:    abs,
		Name:    filepath.Base(abs),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}
}

// Read returns the content of the markdown file at path.
func (m *Manager) Read(path string) (FileContent, error) {
	abs, err := m.resolver.Resolve(path)
	if err != nil {
		return FileContent{}, err
	}
	if !IsMarkdown(abs) {
		return FileContent{}, ErrNotMarkdown
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return FileContent{}, ErrNotFound
		}
		return FileContent{}, err
	}
	if fi.IsDir() {
		return FileContent{}, ErrIsDir
	}
	if fi.Size() > MaxFileSize {
		return FileContent{}, ErrTooLarge
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{FileInfo: fileInfoOf(abs, fi), Content: string(data)}, nil
}

// Save overwrites an existing markdown file with content.
func (m *Manager) Save(path, content string) (FileInfo, error) {
	abs, err := m.resolver.Resolve(path)
	if err != nil {
		return FileInfo{}, err
	}
	if !IsMarkdown(abs) {
		return FileInfo{}, ErrNotMarkdown
	}
	if len(content) > MaxFileSize {
		return FileInfo{}, ErrTooLarge
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, ErrNotFound
		}
		return FileInfo{}, err
	}
	if fi.IsDir() {
		return FileInfo{}, ErrIsDir
	}
	if err := writeFileAtomic(abs, content); err != nil {
		return FileInfo{}, err
	}
	return m.infoOf(abs)
}

// Create writes a new markdown file, creating parent directories as needed.
// It fails with ErrExists if the file already exists.
func (m *Manager) Create(path, content string) (FileInfo, error) {
	abs, err := m.resolver.Resolve(path)
	if err != nil {
		return FileInfo{}, err
	}
	if !IsMarkdown(abs) {
		return FileInfo{}, ErrNotMarkdown
	}
	if len(content) > MaxFileSize {
		return FileInfo{}, ErrTooLarge
	}
	if _, err := os.Stat(abs); err == nil {
		return FileInfo{}, ErrExists
	} else if !os.IsNotExist(err) {
		return FileInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return FileInfo{}, err
	}
	if err := writeFileAtomic(abs, content); err != nil {
		return FileInfo{}, err
	}
	return m.infoOf(abs)
}

// Delete removes a markdown file. Directories and non-markdown files cannot be
// deleted through this method.
func (m *Manager) Delete(path string) error {
	abs, err := m.resolver.Resolve(path)
	if err != nil {
		return err
	}
	if !IsMarkdown(abs) {
		return ErrNotMarkdown
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	if fi.IsDir() {
		return ErrIsDir
	}
	return os.Remove(abs)
}

// Rename moves a markdown file from one path to another. Both paths must be
// markdown files and the destination must not already exist.
func (m *Manager) Rename(from, to string) (FileInfo, error) {
	absFrom, err := m.resolver.Resolve(from)
	if err != nil {
		return FileInfo{}, err
	}
	absTo, err := m.resolver.Resolve(to)
	if err != nil {
		return FileInfo{}, err
	}
	if !IsMarkdown(absFrom) || !IsMarkdown(absTo) {
		return FileInfo{}, ErrNotMarkdown
	}
	fi, err := os.Stat(absFrom)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, ErrNotFound
		}
		return FileInfo{}, err
	}
	if fi.IsDir() {
		return FileInfo{}, ErrIsDir
	}
	if _, err := os.Stat(absTo); err == nil {
		return FileInfo{}, ErrExists
	} else if !os.IsNotExist(err) {
		return FileInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTo), 0o750); err != nil {
		return FileInfo{}, err
	}
	if err := os.Rename(absFrom, absTo); err != nil {
		return FileInfo{}, err
	}
	return m.infoOf(absTo)
}

// Mkdir creates a new directory, including any missing parents.
func (m *Manager) Mkdir(path string) error {
	abs, err := m.resolver.Resolve(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err == nil {
		return ErrExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(abs, 0o750)
}

func (m *Manager) infoOf(abs string) (FileInfo, error) {
	fi, err := os.Stat(abs)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoOf(abs, fi), nil
}

// writeFileAtomic writes content to a temporary file in the target directory
// and renames it into place, so readers never observe a partial write.
func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".mdtree-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
