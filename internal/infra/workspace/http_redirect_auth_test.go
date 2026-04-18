package workspace

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
)

func TestCloneDoesNotForwardCredentialsAcrossHostRedirect(t *testing.T) {
	t.Parallel()

	originAuth, targetAuth, repositoryURL := newRedirectingGitHTTPRepository(t)
	expectedAuth := basicAuthHeader("x-access-token", "ghu_test")

	repoPath := filepath.Join(t.TempDir(), "clone")
	repository, err := cloneOrOpenRepository(context.Background(), repoPath, RepoRequest{
		Name:          "backend",
		RepositoryURL: repositoryURL,
		DefaultBranch: "main",
		HTTPBasicAuth: &HTTPBasicAuthRequest{
			Username: "x-access-token",
			Password: "ghu_test",
		},
	})
	if err != nil {
		t.Fatalf("cloneOrOpenRepository() error = %v", err)
	}
	if repository == nil {
		t.Fatal("cloneOrOpenRepository() returned nil repository")
	}
	if got := originAuth.FirstAuthorization(); got != expectedAuth {
		t.Fatalf("origin authorization = %q, want %q", got, expectedAuth)
	}
	if got := targetAuth.FirstAuthorization(); got != "" {
		t.Fatalf("redirect target authorization = %q, want empty", got)
	}
}

func TestFetchDoesNotForwardCredentialsAcrossHostRedirect(t *testing.T) {
	t.Parallel()

	originAuth, targetAuth, repositoryURL := newRedirectingGitHTTPRepository(t)
	expectedAuth := basicAuthHeader("x-access-token", "ghu_test")

	repository, err := git.PlainInit(filepath.Join(t.TempDir(), "fetch"), false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repositoryURL},
	}); err != nil {
		t.Fatalf("CreateRemote() error = %v", err)
	}

	err = fetchRepository(context.Background(), repository, RepoRequest{
		Name:          "backend",
		RepositoryURL: repositoryURL,
		DefaultBranch: "main",
		HTTPBasicAuth: &HTTPBasicAuthRequest{
			Username: "x-access-token",
			Password: "ghu_test",
		},
	})
	if err != nil {
		t.Fatalf("fetchRepository() error = %v", err)
	}
	if got := originAuth.FirstAuthorization(); got != expectedAuth {
		t.Fatalf("origin authorization = %q, want %q", got, expectedAuth)
	}
	if got := targetAuth.FirstAuthorization(); got != "" {
		t.Fatalf("redirect target authorization = %q, want empty", got)
	}
}

func newRedirectingGitHTTPRepository(t *testing.T) (*requestRecord, *requestRecord, string) {
	t.Helper()

	projectRoot := t.TempDir()
	bareRepoPath := filepath.Join(projectRoot, "private-repo.git")
	sourceRepoPath, _ := createRemoteRepo(t, "main", map[string]string{
		"README.md": "redirect auth coverage",
	})
	runTestCommand(t, exec.Command("git", "clone", "--bare", sourceRepoPath, bareRepoPath))

	targetAuth := &requestRecord{}
	targetListener, targetPort := listenLoopback(t)
	targetMux := http.NewServeMux()
	targetMux.Handle("/", recordAuthorization(targetAuth, &cgi.Handler{
		Path: filepath.Join(strings.TrimSpace(string(commandOutput(t, exec.Command("git", "--exec-path")))), "git-http-backend"),
		Env: []string{
			"GIT_HTTP_EXPORT_ALL=true",
			fmt.Sprintf("GIT_PROJECT_ROOT=%s", projectRoot),
		},
	}))
	targetServer := serveHTTP(t, targetListener, targetMux)
	t.Cleanup(func() {
		_ = targetServer.Close()
	})

	originAuth := &requestRecord{}
	originListener, originPort := listenLoopback(t)
	originRepositoryURL := fmt.Sprintf("http://localhost:%d/private-repo.git", originPort)
	targetRepositoryURL := fmt.Sprintf("http://127.0.0.1:%d/private-repo.git", targetPort)
	originMux := http.NewServeMux()
	originMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		originAuth.Record(r.Header.Get("Authorization"))
		if strings.HasSuffix(r.URL.Path, "/info/refs") {
			http.Redirect(w, r, targetRepositoryURL+r.URL.RequestURI()[len("/private-repo.git"):], http.StatusFound)
			return
		}
		http.Error(w, "unexpected request to redirect origin", http.StatusBadGateway)
	})
	originServer := serveHTTP(t, originListener, originMux)
	t.Cleanup(func() {
		_ = originServer.Close()
	})

	return originAuth, targetAuth, originRepositoryURL
}

func serveHTTP(t *testing.T, listener net.Listener, handler http.Handler) *http.Server {
	t.Helper()

	server := &http.Server{Handler: handler}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return server
}

func listenLoopback(t *testing.T) (net.Listener, int) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	return listener, listener.Addr().(*net.TCPAddr).Port
}

func recordAuthorization(record *requestRecord, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record.Record(r.Header.Get("Authorization"))
		next.ServeHTTP(w, r)
	})
}

func basicAuthHeader(username string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func commandOutput(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s output error = %v", strings.Join(cmd.Args, " "), err)
	}
	return output
}

func runTestCommand(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(cmd.Args, " "), err, string(output))
	}
}

type requestRecord struct {
	mu    sync.Mutex
	first string
}

func (r *requestRecord) Record(value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.first == "" {
		r.first = value
	}
}

func (r *requestRecord) FirstAuthorization() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.first
}
