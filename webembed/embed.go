// Package webembed embeds the compiled frontend (web/dist) into the Go
// binary so production hosts need no Node.js (spec sections 3.2, 72).
package webembed

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the embedded SPA with index.html fallback for client-side
// routes. Asset content is served via http.FileServer with nosniff already
// applied by the API security middleware when mounted under it.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve real files; everything else falls back to index.html.
		path := r.URL.Path
		if path != "/" {
			if _, err := fs.Stat(sub, trimSlash(path)); err != nil {
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}

func trimSlash(p string) string {
	if len(p) > 0 && p[0] == '/' {
		return p[1:]
	}
	return p
}
