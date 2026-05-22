package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsMarkdown(t *testing.T) {
	cases := map[string]bool{
		"a.md": true, "b.markdown": true, "c.MD": true, "d.mkd": true,
		"e.txt": false, "f": false, "g.mdx": false, "h.go": false,
	}
	for name, want := range cases {
		if got := IsMarkdown(name); got != want {
			t.Errorf("IsMarkdown(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestResolverWithinRoot(t *testing.T) {
	root := t.TempDir()
	r := NewResolver(root, false)
	want := filepath.Join(root, "notes", "x.md")
	got, err := r.Resolve(want)
	if err != nil {
		t.Fatalf("Resolve inside root: %v", err)
	}
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

func TestResolverRejectsTraversal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(root, false)
	for _, p := range []string{
		filepath.Join(root, "..", "escape.md"),
		"/etc/passwd",
		root + "/../../etc/passwd",
	} {
		if _, err := r.Resolve(p); err != ErrOutsideRoot {
			t.Errorf("Resolve(%q) error = %v, want ErrOutsideRoot", p, err)
		}
	}
}

func TestResolverRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "vault")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	r := NewResolver(root, false)
	if _, err := r.Resolve(filepath.Join(link, "x.md")); err != ErrOutsideRoot {
		t.Errorf("symlink escape error = %v, want ErrOutsideRoot", err)
	}
}

func TestResolverRejectsEmptyPath(t *testing.T) {
	r := NewResolver(t.TempDir(), false)
	if _, err := r.Resolve(""); err != ErrInvalidPath {
		t.Errorf("Resolve(\"\") = %v, want ErrInvalidPath", err)
	}
}
