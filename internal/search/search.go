package search

import (
	"sort"
	"strings"
)

// DefaultLimit is the default number of results returned by Search.
const DefaultLimit = 50

// Result is one search hit, scored so that higher is a better match.
type Result struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Score int    `json:"score"`
}

// Search returns up to limit markdown files whose name or path matches query,
// ranked best-first. An empty query yields no results.
func (idx *Index) Search(query string, limit int) []Result {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = DefaultLimit
	}

	idx.mu.RLock()
	results := make([]Result, 0, 64)
	for _, d := range idx.docs {
		if score, ok := scoreDoc(q, d); ok {
			results = append(results, Result{Name: d.name, Path: d.path, Score: score})
		}
	}
	idx.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if len(results[i].Path) != len(results[j].Path) {
			return len(results[i].Path) < len(results[j].Path)
		}
		return results[i].Path < results[j].Path
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// scoreDoc ranks a document against a lowercased query. The boolean result
// reports whether the document matched at all.
//
// Match tiers, best to worst: exact name, name prefix, name substring, path
// substring, fuzzy subsequence on name, fuzzy subsequence on path. Longer
// matches are penalised slightly so concise names rank higher.
func scoreDoc(q string, d doc) (int, bool) {
	switch {
	case d.lowerName == q:
		return 1000, true
	case strings.HasPrefix(d.lowerName, q):
		return 850 - lengthPenalty(d.lowerName), true
	case strings.Contains(d.lowerName, q):
		return 650 - strings.Index(d.lowerName, q) - lengthPenalty(d.lowerName), true
	case strings.Contains(d.lowerPath, q):
		return 450 - lengthPenalty(d.lowerPath), true
	case subsequence(q, d.lowerName):
		return 300 - lengthPenalty(d.lowerName), true
	case subsequence(q, d.lowerPath):
		return 150 - lengthPenalty(d.lowerPath), true
	default:
		return 0, false
	}
}

// lengthPenalty scales a string length into a small ranking penalty.
func lengthPenalty(s string) int {
	p := len(s) / 4
	if p > 120 {
		p = 120
	}
	return p
}

// subsequence reports whether the runes of needle appear in order (not
// necessarily contiguously) within haystack.
func subsequence(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	nr := []rune(needle)
	i := 0
	for _, hr := range haystack {
		if hr == nr[i] {
			i++
			if i == len(nr) {
				return true
			}
		}
	}
	return false
}
