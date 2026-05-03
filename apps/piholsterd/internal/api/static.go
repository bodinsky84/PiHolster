package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var staticFiles embed.FS

// StaticHandler serves the embedded SvelteKit build. Unknown paths fall back
// to index.html so that client-side routing works without a 404.
//
// Cache strategy:
//   - _app/** gets a long-lived immutable header because SvelteKit content-hashes
//     every asset filename in that directory.
//   - index.html (and anything outside _app/) gets no-cache so clients always
//     revalidate and pick up new deploys.
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		// dist/ is guaranteed to exist via go:embed; this path is unreachable
		// in normal operation.
		panic("static: failed to sub embed.FS: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/_app/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		// Probe whether the requested path exists in the embedded FS. If it
		// doesn't, rewrite to index.html so SvelteKit handles routing.
		f, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Serve index.html for all unknown paths (SPA fallback).
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r2)
	})
}
