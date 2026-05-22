package search

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAndSearch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "x")
	writeFile(t, root, "docs/guide.md", "x")
	writeFile(t, root, "docs/api/reference.md", "x")
	writeFile(t, root, "notes/todo.md", "x")
	writeFile(t, root, "notes/scratch.txt", "x") // non-markdown: ignored
	writeFile(t, root, ".git/config.md", "x")    // hidden dir: skipped

	idx := NewIndex(root, []string{"node_modules"}, 0, nil)
	count, _, err := idx.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if count != 4 {
		t.Fatalf("indexed %d files, want 4", count)
	}

	results := idx.Search("reference", 10)
	if len(results) == 0 || filepath.Base(results[0].Path) != "reference.md" {
		t.Errorf("search 'reference' top hit = %+v", results)
	}

	// Fuzzy subsequence: "gd" is a subsequence of "guide.md".
	if got := idx.Search("gd", 10); len(got) == 0 {
		t.Errorf("fuzzy search 'gd' returned no results")
	}

	if got := idx.Search("", 10); got != nil {
		t.Errorf("empty query should return nil, got %v", got)
	}
}

func TestIndexMutations(t *testing.T) {
	root := t.TempDir()
	idx := NewIndex(root, nil, 0, nil)

	idx.Add(filepath.Join(root, "a.md"))
	if len(idx.Search("a", 10)) != 1 {
		t.Errorf("after Add, search 'a' should find 1 file")
	}

	idx.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	if len(idx.Search("b", 10)) != 1 {
		t.Errorf("after Rename, search 'b' should find 1 file")
	}

	idx.Remove(filepath.Join(root, "b.md"))
	if len(idx.Search("b", 10)) != 0 {
		t.Errorf("after Remove, search 'b' should find 0 files")
	}
}

func TestRankingPrefersExactName(t *testing.T) {
	root := t.TempDir()
	idx := NewIndex(root, nil, 0, nil)
	idx.Add(filepath.Join(root, "deep/nested/notes.md"))
	idx.Add(filepath.Join(root, "notes.md"))

	results := idx.Search("notes.md", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Path != filepath.Join(root, "notes.md") {
		t.Errorf("exact, shorter path should rank first, got %q", results[0].Path)
	}
}

func TestSubsequence(t *testing.T) {
	cases := []struct {
		needle, hay string
		want        bool
	}{
		{"abc", "aXbXc", true},
		{"abc", "acb", false},
		{"", "anything", true},
		{"go", "golang", true},
		{"xyz", "ab", false},
	}
	for _, c := range cases {
		if got := subsequence(c.needle, c.hay); got != c.want {
			t.Errorf("subsequence(%q, %q) = %v, want %v", c.needle, c.hay, got, c.want)
		}
	}
}
