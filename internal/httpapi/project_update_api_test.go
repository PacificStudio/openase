package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/config"
	projectupdatedomain "github.com/BetterAndBetterII/openase/internal/domain/projectupdate"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestProjectUpdateRoutesCRUDAndRevisions(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	projectUpdateSvc := projectupdateservice.NewService(
		client,
		activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, bus),
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithProjectUpdateService(projectUpdateSvc),
	)

	ctx := context.Background()
	projectID := seedHTTPProject(ctx, t, client)

	createThreadResp := struct {
		Thread projectUpdateThreadResponse `json:"thread"`
	}{}
	executeJSONWithWriteActor(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/updates", projectID),
		map[string]any{
			"status": "on_track",
			"body":   "Initial launch window is green.",
		},
		"user:codex",
		http.StatusCreated,
		&createThreadResp,
	)
	if createThreadResp.Thread.Status != "on_track" || createThreadResp.Thread.CreatedBy != "user:codex" {
		t.Fatalf("create thread response = %+v", createThreadResp.Thread)
	}
	if createThreadResp.Thread.Title != "Initial launch window is green." || createThreadResp.Thread.BodyMarkdown != "Initial launch window is green." {
		t.Fatalf("create thread derived title response = %+v", createThreadResp.Thread)
	}

	secondThreadResp := struct {
		Thread projectUpdateThreadResponse `json:"thread"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/updates", projectID),
		map[string]any{
			"status": "off_track",
			"title":  "Infra migration",
			"body":   "Waiting on cleanup work.",
		},
		http.StatusCreated,
		&secondThreadResp,
	)

	commentResp := struct {
		Comment projectUpdateCommentResponse `json:"comment"`
	}{}
	executeJSONWithWriteActor(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/updates/%s/comments", projectID, createThreadResp.Thread.ID),
		map[string]any{"body": "Need one more canary before Friday."},
		"user:ops",
		http.StatusCreated,
		&commentResp,
	)
	if commentResp.Comment.CreatedBy != "user:ops" {
		t.Fatalf("create comment response = %+v", commentResp.Comment)
	}

	updateThreadResp := struct {
		Thread projectUpdateThreadResponse `json:"thread"`
	}{}
	executeJSONWithWriteActor(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/projects/%s/updates/%s", projectID, createThreadResp.Thread.ID),
		map[string]any{
			"status":      "at_risk",
			"body":        "Blocked by flaky canary verification.",
			"edit_reason": "status recalibration",
		},
		"user:reviewer",
		http.StatusOK,
		&updateThreadResp,
	)
	if updateThreadResp.Thread.Status != "at_risk" || updateThreadResp.Thread.EditCount != 1 {
		t.Fatalf("update thread response = %+v", updateThreadResp.Thread)
	}
	if updateThreadResp.Thread.Title != "Blocked by flaky canary verification." || updateThreadResp.Thread.BodyMarkdown != "Blocked by flaky canary verification." {
		t.Fatalf("update thread derived title response = %+v", updateThreadResp.Thread)
	}

	updateCommentResp := struct {
		Comment projectUpdateCommentResponse `json:"comment"`
	}{}
	executeJSONWithWriteActor(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf(
			"/api/v1/projects/%s/updates/%s/comments/%s",
			projectID,
			createThreadResp.Thread.ID,
			commentResp.Comment.ID,
		),
		map[string]any{
			"body":        "Need one more canary before Friday noon.",
			"edit_reason": "tightened timing",
		},
		"user:ops",
		http.StatusOK,
		&updateCommentResp,
	)
	if updateCommentResp.Comment.EditCount != 1 {
		t.Fatalf("update comment response = %+v", updateCommentResp.Comment)
	}

	threadRevisionsResp := struct {
		Revisions []projectUpdateThreadRevisionResponse `json:"revisions"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/updates/%s/revisions", projectID, createThreadResp.Thread.ID),
		nil,
		http.StatusOK,
		&threadRevisionsResp,
	)
	if len(threadRevisionsResp.Revisions) != 2 || threadRevisionsResp.Revisions[1].Status != "at_risk" {
		t.Fatalf("thread revisions = %+v", threadRevisionsResp.Revisions)
	}

	commentRevisionsResp := struct {
		Revisions []projectUpdateCommentRevisionResponse `json:"revisions"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf(
			"/api/v1/projects/%s/updates/%s/comments/%s/revisions",
			projectID,
			createThreadResp.Thread.ID,
			commentResp.Comment.ID,
		),
		nil,
		http.StatusOK,
		&commentRevisionsResp,
	)
	if len(commentRevisionsResp.Revisions) != 2 || commentRevisionsResp.Revisions[1].BodyMarkdown != "Need one more canary before Friday noon." {
		t.Fatalf("comment revisions = %+v", commentRevisionsResp.Revisions)
	}

	deleteCommentResp := struct {
		DeletedCommentID string `json:"deleted_comment_id"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf(
			"/api/v1/projects/%s/updates/%s/comments/%s",
			projectID,
			createThreadResp.Thread.ID,
			commentResp.Comment.ID,
		),
		nil,
		http.StatusOK,
		&deleteCommentResp,
	)
	if deleteCommentResp.DeletedCommentID != commentResp.Comment.ID {
		t.Fatalf("delete comment response = %+v", deleteCommentResp)
	}

	deleteThreadResp := struct {
		DeletedThreadID string `json:"deleted_thread_id"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/projects/%s/updates/%s", projectID, secondThreadResp.Thread.ID),
		nil,
		http.StatusOK,
		&deleteThreadResp,
	)
	if deleteThreadResp.DeletedThreadID != secondThreadResp.Thread.ID {
		t.Fatalf("delete thread response = %+v", deleteThreadResp)
	}

	listResp := struct {
		Threads    []projectUpdateThreadResponse `json:"threads"`
		NextCursor string                        `json:"next_cursor"`
		HasMore    bool                          `json:"has_more"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/updates", projectID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Threads) != 2 {
		t.Fatalf("list threads len = %d, want 2", len(listResp.Threads))
	}
	if listResp.HasMore || listResp.NextCursor != "" {
		t.Fatalf("list threads pagination = %+v", listResp)
	}
	if listResp.Threads[0].ID != secondThreadResp.Thread.ID || !listResp.Threads[0].IsDeleted {
		t.Fatalf("list threads[0] = %+v", listResp.Threads[0])
	}
	if listResp.Threads[0].Title != "Infra migration" || listResp.Threads[0].BodyMarkdown != "Waiting on cleanup work." {
		t.Fatalf("list threads[0] explicit title/body = %+v", listResp.Threads[0])
	}
	if listResp.Threads[1].CommentCount != 0 || len(listResp.Threads[1].Comments) != 1 || !listResp.Threads[1].Comments[0].IsDeleted {
		t.Fatalf("list threads[1] = %+v", listResp.Threads[1])
	}
}

func TestProjectUpdateRoutesSupportCursorPagination(t *testing.T) {
	client := openTestEntClient(t)
	projectUpdateSvc := projectupdateservice.NewService(client, nil)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithProjectUpdateService(projectUpdateSvc),
	)

	ctx := context.Background()
	projectID := seedHTTPProject(ctx, t, client)
	sharedActivityAt := time.Date(2026, 4, 1, 11, 30, 0, 0, time.UTC)
	olderActivityAt := sharedActivityAt.Add(-1 * time.Hour)

	createThread := func(status, title, body string) string {
		resp := struct {
			Thread projectUpdateThreadResponse `json:"thread"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%s/updates", projectID),
			map[string]any{
				"status": status,
				"title":  title,
				"body":   body,
			},
			http.StatusCreated,
			&resp,
		)
		return resp.Thread.ID
	}

	threadAID := createThread("on_track", "Thread A", "Alpha")
	threadBID := createThread("at_risk", "Thread B", "Beta")
	threadCID := createThread("off_track", "Thread C", "Gamma")

	threadAUUID := uuid.MustParse(threadAID)
	threadBUUID := uuid.MustParse(threadBID)
	threadCUUID := uuid.MustParse(threadCID)

	if _, err := client.ProjectUpdateThread.UpdateOneID(threadAUUID).
		SetLastActivityAt(sharedActivityAt).
		SetUpdatedAt(sharedActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadA activity: %v", err)
	}
	if _, err := client.ProjectUpdateThread.UpdateOneID(threadBUUID).
		SetLastActivityAt(sharedActivityAt).
		SetUpdatedAt(sharedActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadB activity: %v", err)
	}
	if _, err := client.ProjectUpdateThread.UpdateOneID(threadCUUID).
		SetLastActivityAt(olderActivityAt).
		SetUpdatedAt(olderActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadC activity: %v", err)
	}

	firstPage := struct {
		Threads    []projectUpdateThreadResponse `json:"threads"`
		NextCursor string                        `json:"next_cursor"`
		HasMore    bool                          `json:"has_more"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/updates?limit=2", projectID),
		nil,
		http.StatusOK,
		&firstPage,
	)
	if len(firstPage.Threads) != 2 || !firstPage.HasMore || firstPage.NextCursor == "" {
		t.Fatalf("first page = %+v", firstPage)
	}
	if firstPage.Threads[0].LastActivityAt != sharedActivityAt.Format(time.RFC3339Nano) ||
		firstPage.Threads[1].LastActivityAt != sharedActivityAt.Format(time.RFC3339Nano) {
		t.Fatalf("first page shared timestamps = %+v", firstPage.Threads)
	}
	if firstPage.Threads[0].ID < firstPage.Threads[1].ID {
		t.Fatalf("first page ids = %s, %s, want desc tie-break", firstPage.Threads[0].ID, firstPage.Threads[1].ID)
	}

	secondPage := struct {
		Threads    []projectUpdateThreadResponse `json:"threads"`
		NextCursor string                        `json:"next_cursor"`
		HasMore    bool                          `json:"has_more"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf(
			"/api/v1/projects/%s/updates?limit=2&before=%s",
			projectID,
			url.QueryEscape(firstPage.NextCursor),
		),
		nil,
		http.StatusOK,
		&secondPage,
	)
	if len(secondPage.Threads) != 1 || secondPage.Threads[0].ID != threadCID {
		t.Fatalf("second page threads = %+v, want only threadC", secondPage.Threads)
	}
	if secondPage.HasMore || secondPage.NextCursor != "" {
		t.Fatalf("second page pagination = %+v", secondPage)
	}

	badLimit := performJSONRequest(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/updates?limit=0", projectID),
		"",
	)
	assertAPIErrorResponse(
		t,
		badLimit,
		http.StatusBadRequest,
		"INVALID_UPDATES_PAGE",
		"limit must be an integer between 1 and 100",
	)

	badCursor := performJSONRequest(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/updates?before=bad", projectID),
		"",
	)
	assertAPIErrorResponse(
		t,
		badCursor,
		http.StatusBadRequest,
		"INVALID_UPDATES_PAGE",
		"before cursor must be in timestamp|id format",
	)
}

func TestProjectUpdateCreateEmitsActivityStreamEvent(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	projectUpdateSvc := projectupdateservice.NewService(
		client,
		activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, bus),
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithProjectUpdateService(projectUpdateSvc),
	)

	projectID := seedHTTPProject(context.Background(), t, client)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	response, cancel := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/events/stream", projectID))
	defer func() {
		cancel()
		if err := response.Body.Close(); err != nil {
			t.Errorf("close project event bus response body: %v", err)
		}
	}()

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/updates", projectID),
		`{"status":"on_track","body":"Everything is green."}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create thread 201, got %d: %s", rec.Code, rec.Body.String())
	}

	body := readSSEBody(t, response, cancel)
	if !strings.Contains(body, `"topic":"activity.events"`) {
		t.Fatalf("expected activity topic on project event bus, got %q", body)
	}
	if !strings.Contains(body, `"type":"project_update_thread.created"`) {
		t.Fatalf("expected project update create event, got %q", body)
	}
	if !strings.Contains(body, `"thread_title":"Everything is green."`) {
		t.Fatalf("expected thread title metadata in stream, got %q", body)
	}
}

func TestProjectUpdateRequestParsersAndErrors(t *testing.T) {
	projectID := uuid.New()
	threadID := uuid.New()
	commentID := uuid.New()
	editReason := " refined wording "

	createInput, err := parseCreateProjectUpdateThreadRequest(projectID, "user:codex", rawCreateProjectUpdateThreadRequest{
		Status: "at_risk",
		Body:   "  Investigating blockers  ",
	})
	if err != nil {
		t.Fatalf("parseCreateProjectUpdateThreadRequest() error = %v", err)
	}
	if createInput.Status != projectupdateservice.StatusAtRisk || createInput.Title != "" || createInput.Body != "Investigating blockers" || createInput.CreatedBy != "user:codex" {
		t.Fatalf("parseCreateProjectUpdateThreadRequest() = %+v", createInput)
	}

	title := "  Delivery  "
	createInputWithTitle, err := parseCreateProjectUpdateThreadRequest(projectID, "user:codex", rawCreateProjectUpdateThreadRequest{
		Status: "at_risk",
		Title:  &title,
		Body:   "  Investigating blockers  ",
	})
	if err != nil {
		t.Fatalf("parseCreateProjectUpdateThreadRequest(with title) error = %v", err)
	}
	if createInputWithTitle.Title != "Delivery" {
		t.Fatalf("parseCreateProjectUpdateThreadRequest(with title) = %+v", createInputWithTitle)
	}

	updateInput, err := parseUpdateProjectUpdateCommentRequest(projectID, threadID, commentID, "user:reviewer", rawUpdateProjectUpdateCommentRequest{
		Body:       "  Updated body  ",
		EditReason: &editReason,
	})
	if err != nil {
		t.Fatalf("parseUpdateProjectUpdateCommentRequest() error = %v", err)
	}
	if updateInput.Body != "Updated body" || updateInput.EditedBy != "user:reviewer" || updateInput.EditReason != "refined wording" {
		t.Fatalf("parseUpdateProjectUpdateCommentRequest() = %+v", updateInput)
	}

	pageInput, err := parseListProjectUpdatesPageRequest(projectID, projectupdatedomain.ListThreadsPageRequest{
		Limit:  " 5 ",
		Before: " 2026-04-01T10:00:00Z|" + threadID.String() + " ",
	})
	if err != nil {
		t.Fatalf("parseListProjectUpdatesPageRequest() error = %v", err)
	}
	if pageInput.ProjectID != projectID || pageInput.Limit != 5 || pageInput.Before == nil {
		t.Fatalf("parseListProjectUpdatesPageRequest() = %+v", pageInput)
	}

	if _, err := parseCreateProjectUpdateThreadRequest(projectID, "", rawCreateProjectUpdateThreadRequest{
		Status: "bad",
		Body:   "y",
	}); err == nil {
		t.Fatal("expected invalid status error")
	}
	if _, err := parseCreateProjectUpdateCommentRequest(projectID, threadID, "", rawCreateProjectUpdateCommentRequest{
		Body: " ",
	}); err == nil {
		t.Fatal("expected empty comment body error")
	}
	if _, err := parseListProjectUpdatesPageRequest(projectID, projectupdatedomain.ListThreadsPageRequest{
		Limit: "500",
	}); err == nil {
		t.Fatal("expected invalid limit error")
	}
	if _, err := parseListProjectUpdatesPageRequest(projectID, projectupdatedomain.ListThreadsPageRequest{
		Before: "bad",
	}); err == nil {
		t.Fatal("expected invalid cursor error")
	}

	for _, testCase := range []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{err: projectupdateservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
		{err: projectupdateservice.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
		{err: projectupdateservice.ErrThreadNotFound, wantStatus: http.StatusNotFound, wantCode: "UPDATE_THREAD_NOT_FOUND"},
		{err: projectupdateservice.ErrCommentNotFound, wantStatus: http.StatusNotFound, wantCode: "UPDATE_COMMENT_NOT_FOUND"},
	} {
		rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
			return writeProjectUpdateError(c, testCase.err)
		})
		assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
	}
}

func seedHTTPProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug(fmt.Sprintf("platform-%s", uuid.NewString()[:8])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project.ID
}
