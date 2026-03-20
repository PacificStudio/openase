package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
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
