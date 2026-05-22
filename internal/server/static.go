package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// frontendMissingHTML is served when the binary was built without a compiled
// frontend (the embedded dist directory contains no index.html).
const frontendMissingHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>mdtree</title>
<style>body{font:16px/1.6 system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1.5rem;color:#1a1a1a}code{background:#eee;padding:.15em .4em;border-radius:4px}</style>
</head><body>
<h1>mdtree</h1>
<p>The backend is running, but the frontend has not been built into this binary.</p>
<p>Build it and rebuild the binary:</p>
<pre><code>npm --prefix web install
npm --prefix web run build
go build ./cmd/mdtree</code></pre>
<p>Or run <code>scripts/build.sh</code>.</p>
</body></html>`

// spaHandler serves the embedded single-page application. Requests for paths
// that do not match a static asset fall back to index.html so client-side
// routing survives a deep link or page refresh. When no frontend was built in,
// every request gets a friendly explanatory page instead.
func spaHandler(dist fs.FS) http.Handler {
	_, statErr := fs.Stat(dist, "index.html")
	frontendBuilt := statErr == nil
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !frontendBuilt {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(frontendMissingHTML))
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "." {
			name = "index.html"
		}
		if _, err := fs.Stat(dist, name); err != nil {
			// Unknown path: serve the SPA shell for client-side routing.
			req := r.Clone(r.Context())
			req.URL.Path = "/"
			fileServer.ServeHTTP(w, req)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
