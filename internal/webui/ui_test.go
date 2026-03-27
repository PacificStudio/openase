package webui

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestHandlerForFSMissingAssets(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handlerForFS(fstest.MapFS{
		".keep": {Data: []byte("placeholder\n")},
	}).ServeHTTP(rec, req)

	res := rec.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusServiceUnavailable)
	}

	if contentType := res.Header.Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", contentType, "text/plain; charset=utf-8")
	}

	if !strings.Contains(string(body), "OpenASE web UI assets are not built") {
		t.Fatalf("body = %q, want missing assets guidance", string(body))
	}
}

func TestHandlerForFSServesIndexAndSPAPath(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"index.html":  {Data: []byte("index")},
		"ticket.html": {Data: []byte("ticket")},
	}

	testCases := []struct {
		name string
		path string
		want string
	}{
		{name: "root", path: "/", want: "index"},
		{name: "static file", path: "/ticket.html", want: "ticket"},
		{name: "spa route", path: "/tickets/ASE-47", want: "index"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			handlerForFS(fsys).ServeHTTP(rec, req)

			res := rec.Result()
			defer func() {
				_ = res.Body.Close()
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
			}

			if got := string(body); got != tc.want {
				t.Fatalf("body = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandlerUsesEmbeddedAssetsWithoutPanic(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Result().StatusCode == 0 {
		t.Fatal("Handler() should produce an HTTP response")
	}
}

func TestHandlerForFSReturnsNotFoundForMissingExtensionPath(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/missing.js", nil)
	rec := httptest.NewRecorder()

	handlerForFS(fstest.MapFS{
		"index.html": {Data: []byte("index")},
	}).ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Result().StatusCode, http.StatusNotFound)
	}
}

func TestServeFileFallbackBranches(t *testing.T) {
	t.Parallel()

	t.Run("stat error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/broken.txt", nil)
		rec := httptest.NewRecorder()

		serveFile(statErrorFS{}, rec, req, "broken.txt")

		if rec.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Result().StatusCode, http.StatusNotFound)
		}
	})

	t.Run("non read seeker", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/note.txt", nil)
		rec := httptest.NewRecorder()

		serveFile(nonReadSeekerFS{}, rec, req, "note.txt")

		res := rec.Result()
		defer func() {
			_ = res.Body.Close()
		}()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("read response body: %v", err)
		}
		if res.StatusCode != http.StatusOK || string(body) != "note" {
			t.Fatalf("status/body = %d/%q, want %d/%q", res.StatusCode, string(body), http.StatusOK, "note")
		}
	})
}

type statErrorFS struct{}

func (statErrorFS) Open(string) (fs.File, error) {
	return statErrorFile{}, nil
}

type statErrorFile struct{}

func (statErrorFile) Stat() (fs.FileInfo, error) { return nil, errors.New("boom") }
func (statErrorFile) Read([]byte) (int, error)   { return 0, io.EOF }
func (statErrorFile) Close() error               { return nil }

type nonReadSeekerFS struct{}

func (nonReadSeekerFS) Open(string) (fs.File, error) {
	return &nonReadSeekerFile{reader: strings.NewReader("note")}, nil
}

type nonReadSeekerFile struct {
	reader *strings.Reader
}

func (f *nonReadSeekerFile) Stat() (fs.FileInfo, error) {
	return testFileInfo{name: "note.txt", size: 4}, nil
}
func (f *nonReadSeekerFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *nonReadSeekerFile) Close() error               { return nil }

type testFileInfo struct {
	name string
	size int64
}

func (i testFileInfo) Name() string       { return i.name }
func (i testFileInfo) Size() int64        { return i.size }
func (i testFileInfo) Mode() fs.FileMode  { return 0 }
func (i testFileInfo) ModTime() time.Time { return time.Time{} }
func (i testFileInfo) IsDir() bool        { return false }
func (i testFileInfo) Sys() any           { return nil }
