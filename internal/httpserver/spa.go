package httpserver

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/webui"
)

// registerSPA serves the Next.js static export embedded via internal/webui.
// It must be registered last: every API route above takes priority.
//
// Next's static export writes each route as <path>.html (e.g.
// users/detail.html for /users/detail), not <path>/index.html, so a plain
// http.FileServer's automatic index-file handling doesn't apply here; the
// exact lookup order below (path, path.html, path/index.html, then the
// root index.html as an SPA fallback) is what makes a hard refresh on a
// nested route like /users/detail actually work.
func (s *Server) registerSPA() error {
	root, err := webui.FS()
	if err != nil {
		return err
	}

	s.app.Use(func(c *fiber.Ctx) error {
		p := c.Path()
		for _, prefix := range apiPathPrefixes {
			if strings.HasPrefix(p, prefix) {
				return c.Next()
			}
		}
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.Next()
		}

		clean := strings.TrimPrefix(p, "/")
		for _, candidate := range candidatePaths(clean) {
			data, err := fs.ReadFile(root, candidate)
			if err == nil {
				c.Type(extOf(candidate))
				return c.Send(data)
			}
		}

		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		c.Type("html")
		return c.Send(data)
	})
	return nil
}

func candidatePaths(clean string) []string {
	if clean == "" {
		return []string{"index.html"}
	}
	if strings.Contains(clean, ".") {
		// already has an extension (js, css, ico, svg, ...): serve as-is.
		return []string{clean}
	}
	return []string{clean, clean + ".html", clean + "/index.html"}
}

func extOf(path string) string {
	i := strings.LastIndex(path, ".")
	if i == -1 {
		return "html"
	}
	return path[i+1:]
}

var apiPathPrefixes = []string{
	"/api/", "/oauth2/", "/external/", "/device/", "/forwardauth/",
	"/.well-known/", "/healthz", "/metrics",
}
