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

const missingAssetsMessage = "OpenASE web UI assets are not built. Run `corepack pnpm --dir web install --frozen-lockfile && corepack pnpm --dir web run build` (or `make build`) before rebuilding the Go binary."

func Handler() http.Handler {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}

	return handlerForFS(staticFS)
}

func handlerForFS(staticFS fs.FS) http.Handler {
	hasIndex := fileExists(staticFS, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hasIndex {
			serveMissingAssets(w)
			return
		}

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

func fileExists(fsys fs.FS, name string) bool {
	_, err := fs.Stat(fsys, name)
	return err == nil
}

func serveMissingAssets(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, missingAssetsMessage+"\n")
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
