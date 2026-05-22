package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/SinlinLi/mdtree/internal/api"
	"github.com/SinlinLi/mdtree/internal/auth"
	"github.com/SinlinLi/mdtree/internal/files"
	"github.com/SinlinLi/mdtree/internal/metrics"
	"github.com/SinlinLi/mdtree/internal/search"
)

const testPassword = "test-password-1234"

// newTestServer builds a fully wired mdtree server over a temporary vault.
func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("readme.md", "# Hi\n")
	write("docs/guide.md", "# Guide\n")

	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	resolver := files.NewResolver(root, false)
	sessions := auth.NewSessionStore(time.Hour)
	authn := auth.NewAuthenticator(sessions, false)
	idx := search.NewIndex(root, nil, 0, nil)
	if _, _, err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	met := metrics.New()

	handler := &api.Handler{
		Files:        files.NewManager(resolver),
		Lister:       files.NewLister(resolver, nil, false),
		Index:        idx,
		Auth:         authn,
		Sessions:     sessions,
		Limiter:      auth.NewRateLimiter(50, time.Minute),
		Metrics:      met,
		PasswordHash: hash,
		Root:         root,
		Log:          logger,
	}
	srv := New(Config{
		API:     handler,
		Auth:    authn,
		Metrics: met,
		DistFS:  fstest.MapFS{},
		Log:     logger,
	})
	ts := httptest.NewServer(srv.httpServer.Handler)
	t.Cleanup(ts.Close)
	return ts, root
}

func TestServerIntegration(t *testing.T) {
	ts, root := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	do := func(method, path, body string) (int, string) {
		t.Helper()
		var r io.Reader
		if body != "" {
			r = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, ts.URL+path, r)
		if err != nil {
			t.Fatal(err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		data, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(data)
	}

	// The health check is reachable without authentication.
	if status, _ := do("GET", "/healthz", ""); status != http.StatusOK {
		t.Errorf("healthz status = %d, want 200", status)
	}

	// Protected routes reject requests without a session.
	if status, _ := do("GET", "/api/tree", ""); status != http.StatusUnauthorized {
		t.Errorf("unauthenticated tree status = %d, want 401", status)
	}

	// A wrong password is rejected.
	if status, _ := do("POST", "/api/auth/login", `{"password":"wrong"}`); status != http.StatusUnauthorized {
		t.Errorf("bad-password login status = %d, want 401", status)
	}

	// The correct password establishes a session cookie.
	if status, _ := do("POST", "/api/auth/login", `{"password":"`+testPassword+`"}`); status != http.StatusOK {
		t.Fatalf("login status = %d, want 200", status)
	}

	// The tree lists markdown files.
	if status, body := do("GET", "/api/tree", ""); status != http.StatusOK || !strings.Contains(body, "readme.md") {
		t.Errorf("tree status=%d body=%s", status, body)
	}

	// Create then read back a new file.
	newPath := filepath.Join(root, "fresh.md")
	if status, _ := do("POST", "/api/file", `{"path":"`+newPath+`","content":"# Fresh\n"}`); status != http.StatusCreated {
		t.Errorf("create status = %d, want 201", status)
	}
	if status, body := do("GET", "/api/file?path="+newPath, ""); status != http.StatusOK || !strings.Contains(body, "# Fresh") {
		t.Errorf("read status=%d body=%s", status, body)
	}

	// Search finds an indexed file.
	if status, body := do("GET", "/api/search?q=guide", ""); status != http.StatusOK || !strings.Contains(body, "guide.md") {
		t.Errorf("search status=%d body=%s", status, body)
	}

	// Access outside the root is forbidden.
	if status, _ := do("GET", "/api/file?path=/etc/passwd", ""); status != http.StatusForbidden {
		t.Errorf("traversal status = %d, want 403", status)
	}

	// Logout invalidates the session.
	if status, _ := do("POST", "/api/auth/logout", ""); status != http.StatusOK {
		t.Errorf("logout status = %d, want 200", status)
	}
	if status, _ := do("GET", "/api/tree", ""); status != http.StatusUnauthorized {
		t.Errorf("tree after logout status = %d, want 401", status)
	}
}
