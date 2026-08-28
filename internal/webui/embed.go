// Package webui embeds the Next.js admin console's static export (built
// separately with `npm run build` in web/, output copied here) directly
// into the Go binary. No Node.js server runs in production; this just
// serves pre-built HTML/CSS/JS files, keeping the single-binary
// distribution model. The SPA calls the same JSON API
// (/api/v1, /oauth2, /external/v1, /iot) documented for any other client.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded static export rooted at its "dist" directory,
// ready to hand to a static file server.
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
