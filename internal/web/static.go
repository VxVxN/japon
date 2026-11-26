package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// staticFS carries the web UI assets shipped inside the binary.
//
//go:embed all:static
var staticFS embed.FS

// Static returns the embedded asset tree rooted at the "static" directory.
func Static() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("web: embedded assets missing: " + err.Error())
	}
	return sub
}

// ServeIndex answers GET / with the embedded index.html and lets the page
// pull its css/js from /static/.
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(Static(), "index.html")
	if err != nil {
		http.Error(w, "index.html not embedded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// StaticHandler serves the embedded assets under the given URL prefix
// (e.g. "/static/").
func StaticHandler(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.FS(Static())))
}

// assetContentType maps known extensions so browsers get proper MIME types
// even on exotic setups.
var assetContentType = map[string]string{
	".html": "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript; charset=utf-8",
	".json": "application/json; charset=utf-8",
	".svg":  "image/svg+xml",
}

func contentTypeFor(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		if ct, ok := assetContentType[name[i:]]; ok {
			return ct
		}
	}
	return "application/octet-stream"
}
