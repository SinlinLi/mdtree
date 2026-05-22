// Package web embeds the built mdtree frontend assets into the binary.
package web

import (
	"embed"
	"io/fs"
)

// distFS holds the built frontend. The dist directory is produced by
// `npm run build` (see scripts/build.sh) before `go build`.
//
//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded frontend file system rooted at dist/.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
