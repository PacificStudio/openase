package githubrepo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	domain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
	"github.com/google/uuid"
)

type stubResolver struct {
	token string
	err   error
}

func (s stubResolver) ResolveProjectCredential(context.Context, uuid.UUID) (githubauthdomain.ResolvedCredential, error) {
	if s.err != nil {
		return githubauthdomain.ResolvedCredential{}, s.err
	}
	return githubauthdomain.ResolvedCredential{
		Token: s.token,
		Probe: githubauthdomain.ConfiguredProbe(),
	}, nil
}

func TestListRepositoriesSearchesViaGitHubSearchAPI(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			_, _ = io.WriteString(w, `{"login":"octocat"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/user/orgs":
			_, _ = io.WriteString(w, `[{"login":"acme"},{"login":"platform"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/search/repositories":
			queries = append(queries, r.URL.Query())
			_, _ = io.WriteString(w, `{"items":[{"id":2,"name":"backend","full_name":"acme/backend","private":true,"html_url":"https://github.com/acme/backend","clone_url":"https://github.com/acme/backend.git","default_branch":"develop","visibility":"private","owner":{"login":"acme"}}]}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(stubResolver{token: "ghu_test"}, server.Client()).(*service)
	svc.baseURL = server.URL

	page, err := svc.ListRepositories(context.Background(), domain.ListRepositoriesInput{
		ProjectID: uuid.New(),
		Query:     "back",
		Page:      1,
	})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(page.Repositories) != 1 || page.Repositories[0].FullName != "acme/backend" {
		t.Fatalf("ListRepositories() = %+v", page)
	}
	if page.NextCursor != "" {
		t.Fatalf("ListRepositories() next_cursor = %q, want empty", page.NextCursor)
	}
	if len(queries) != 1 {
		t.Fatalf("search query count = %d, want 1", len(queries))
	}
	if queries[0].Get("page") != "1" || queries[0].Get("sort") != "updated" || queries[0].Get("order") != "desc" {
		t.Fatalf("search query params = %v", queries[0])
	}
	searchQuery := queries[0].Get("q")
	if !strings.Contains(searchQuery, "back") || !strings.Contains(searchQuery, "user:octocat") || !strings.Contains(searchQuery, "org:acme") || !strings.Contains(searchQuery, "org:platform") || !strings.Contains(searchQuery, "fork:true") {
		t.Fatalf("search query = %q", searchQuery)
	}
}

func TestListRepositoriesFallsBackToBrowsableReposWhenSearchReturnsNoMatches(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			_, _ = io.WriteString(w, `{"login":"octocat"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/user/orgs":
			_, _ = io.WriteString(w, `[]`)
		case r.Method == http.MethodGet && r.URL.Path == "/search/repositories":
			_, _ = io.WriteString(w, `{"items":[]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/user/repos" && r.URL.Query().Get("page") == "1":
			_, _ = io.WriteString(w, `[{"id":1193730355,"name":"TodoApp","full_name":"octocat/TodoApp","private":true,"html_url":"https://github.com/octocat/TodoApp","clone_url":"https://github.com/octocat/TodoApp.git","default_branch":"main","visibility":"private","owner":{"login":"octocat"}}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(stubResolver{token: "ghu_test"}, server.Client()).(*service)
	svc.baseURL = server.URL

	page, err := svc.ListRepositories(context.Background(), domain.ListRepositoriesInput{
		ProjectID: uuid.New(),
		Query:     "todo",
		Page:      1,
	})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(page.Repositories) != 1 || page.Repositories[0].FullName != "octocat/TodoApp" {
		t.Fatalf("ListRepositories() = %+v", page)
	}
	if len(requests) < 4 {
		t.Fatalf("requests = %#v, want search plus browse fallback", requests)
	}
	if !strings.Contains(requests[2], "/search/repositories") || !strings.Contains(requests[3], "/user/repos?") {
		t.Fatalf("requests = %#v, want search then user/repos fallback", requests)
	}
}

func TestListRepositoriesBrowsesUserReposWhenQueryEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user/repos" && r.URL.Query().Get("page") == "2":
			w.Header().Set("Link", `<https://example.test/user/repos?page=3>; rel="next"`)
			_, _ = io.WriteString(w, `[{"id":1,"name":"frontend","full_name":"acme/frontend","private":true,"html_url":"https://github.com/acme/frontend","clone_url":"https://github.com/acme/frontend.git","default_branch":"main","visibility":"private","owner":{"login":"acme"}}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(stubResolver{token: "ghu_test"}, server.Client()).(*service)
	svc.baseURL = server.URL

	page, err := svc.ListRepositories(context.Background(), domain.ListRepositoriesInput{
		ProjectID: uuid.New(),
		Query:     "",
		Page:      2,
	})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(page.Repositories) != 1 || page.Repositories[0].FullName != "acme/frontend" {
		t.Fatalf("ListRepositories() = %+v", page)
	}
	if page.NextCursor != "3" {
		t.Fatalf("ListRepositories() next_cursor = %q, want 3", page.NextCursor)
	}
}

func TestListNamespaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, `{"login":"octocat"}`)
		case "/user/orgs":
			_, _ = io.WriteString(w, `[{"login":"acme"},{"login":"platform"}]`)
		default:
			t.Fatalf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := NewService(stubResolver{token: "ghu_test"}, server.Client()).(*service)
	svc.baseURL = server.URL

	namespaces, err := svc.ListNamespaces(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}
	if len(namespaces) != 3 {
		t.Fatalf("ListNamespaces() len = %d, want 3", len(namespaces))
	}
	if namespaces[0].Login != "octocat" || namespaces[0].Kind != domain.NamespaceKindUser {
		t.Fatalf("ListNamespaces() first = %+v", namespaces[0])
	}
}

func TestCreateRepositoryRoutesUserAndOrg(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, `{"login":"octocat"}`)
		case "/user/repos", "/orgs/acme/repos":
			body, _ := io.ReadAll(r.Body)
			requests = append(requests, r.Method+" "+r.URL.Path+" "+strings.TrimSpace(string(body)))
			w.WriteHeader(http.StatusCreated)
			if r.URL.Path == "/user/repos" {
				_, _ = io.WriteString(w, `{"id":1,"name":"tooling","full_name":"octocat/tooling","private":false,"html_url":"https://github.com/octocat/tooling","clone_url":"https://github.com/octocat/tooling.git","default_branch":"main","visibility":"public","owner":{"login":"octocat"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"id":2,"name":"backend","full_name":"acme/backend","private":true,"html_url":"https://github.com/acme/backend","clone_url":"https://github.com/acme/backend.git","default_branch":"main","visibility":"private","owner":{"login":"acme"}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	svc := NewService(stubResolver{token: "ghu_test"}, server.Client()).(*service)
	svc.baseURL = server.URL

	userRepo, err := svc.CreateRepository(context.Background(), domain.CreateRepositoryInput{
		ProjectID:  uuid.New(),
		Owner:      "octocat",
		Name:       "tooling",
		Visibility: domain.VisibilityPublic,
		AutoInit:   true,
	})
	if err != nil {
		t.Fatalf("CreateRepository(user) error = %v", err)
	}
	if userRepo.FullName != "octocat/tooling" {
		t.Fatalf("CreateRepository(user) = %+v", userRepo)
	}

	orgRepo, err := svc.CreateRepository(context.Background(), domain.CreateRepositoryInput{
		ProjectID:  uuid.New(),
		Owner:      "acme",
		Name:       "backend",
		Visibility: domain.VisibilityPrivate,
		AutoInit:   true,
	})
	if err != nil {
		t.Fatalf("CreateRepository(org) error = %v", err)
	}
	if orgRepo.FullName != "acme/backend" {
		t.Fatalf("CreateRepository(org) = %+v", orgRepo)
	}

	if len(requests) != 2 || !strings.Contains(requests[0], "/user/repos") || !strings.Contains(requests[1], "/orgs/acme/repos") {
		t.Fatalf("CreateRepository() requests = %#v", requests)
	}
}
