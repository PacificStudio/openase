package webui

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:static
var assets embed.FS

// Handler serves the embedded web control plane assets.
func Handler() http.Handler {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			requestPath = "index.html"
		}

		if _, err := fs.Stat(staticFS, requestPath); err == nil {
			serveFile(staticFS, w, r, requestPath)
			return
		}

		if !strings.Contains(path.Base(requestPath), ".") {
			serveFile(staticFS, w, r, "index.html")
			return
		}

		http.NotFound(w, r)
	})
}

func serveFile(staticFS fs.FS, w http.ResponseWriter, r *http.Request, name string) {
	file, err := staticFS.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if readSeeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, info.Name(), info.ModTime(), readSeeker)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(data))
}
