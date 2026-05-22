package files

import (
	"os"
	"path/filepath"
	"testing"
)

func newManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	return NewManager(NewResolver(root, false)), root
}

func TestCreateReadSave(t *testing.T) {
	m, root := newManager(t)
	p := filepath.Join(root, "note.md")

	if _, err := m.Create(p, "# Hello\n"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := m.Create(p, "x"); err != ErrExists {
		t.Errorf("Create existing = %v, want ErrExists", err)
	}

	fc, err := m.Read(p)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if fc.Content != "# Hello\n" {
		t.Errorf("content = %q, want %q", fc.Content, "# Hello\n")
	}

	if _, err := m.Save(p, "# Changed\n"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fc, _ = m.Read(p)
	if fc.Content != "# Changed\n" {
		t.Errorf("content after save = %q", fc.Content)
	}
}

func TestSaveMissingFileFails(t *testing.T) {
	m, root := newManager(t)
	if _, err := m.Save(filepath.Join(root, "ghost.md"), "x"); err != ErrNotFound {
		t.Errorf("Save missing file = %v, want ErrNotFound", err)
	}
}

func TestRejectNonMarkdown(t *testing.T) {
	m, root := newManager(t)
	if _, err := m.Create(filepath.Join(root, "a.txt"), "x"); err != ErrNotMarkdown {
		t.Errorf("Create .txt = %v, want ErrNotMarkdown", err)
	}
	if _, err := m.Read(filepath.Join(root, "a.txt")); err != ErrNotMarkdown {
		t.Errorf("Read .txt = %v, want ErrNotMarkdown", err)
	}
}

func TestRenameAndDelete(t *testing.T) {
	m, root := newManager(t)
	from := filepath.Join(root, "a.md")
	to := filepath.Join(root, "sub", "b.md")
	if _, err := m.Create(from, "data"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Rename(from, to); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := os.Stat(to); err != nil {
		t.Errorf("renamed file missing: %v", err)
	}
	if err := m.Delete(to); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(to); !os.IsNotExist(err) {
		t.Errorf("file still present after Delete")
	}
}

func TestAtomicSaveLeavesNoTempFiles(t *testing.T) {
	m, root := newManager(t)
	p := filepath.Join(root, "n.md")
	if _, err := m.Create(p, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Save(p, "v2"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestListMarkdownOnly(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"keep.md", "skip.txt", "also.markdown"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	lister := NewLister(NewResolver(root, false), nil, false)
	listing, err := lister.List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := map[string]string{}
	for _, e := range listing.Entries {
		got[e.Name] = e.Type
	}
	if got["skip.txt"] != "" {
		t.Errorf("non-markdown file should not be listed")
	}
	if got["keep.md"] != "file" || got["also.markdown"] != "file" {
		t.Errorf("markdown files missing from listing: %v", got)
	}
	if got["subdir"] != "dir" {
		t.Errorf("subdirectory should be listed for navigation")
	}
}
