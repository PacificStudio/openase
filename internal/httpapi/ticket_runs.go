package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	ticketRunActivityStreamTopic = provider.MustParseTopic("activity.events")
	ticketRunStreamTopic         = provider.MustParseTopic("ticket.run.events")
	ticketRunLifecycleStreamType = provider.MustParseEventType("ticket.run.lifecycle")
	ticketRunTraceStreamType     = provider.MustParseEventType("ticket.run.trace")
	ticketRunStepStreamType      = provider.MustParseEventType("ticket.run.step")
	ticketRunSummaryStreamType   = provider.MustParseEventType("ticket.run.summary")
)

const ticketRunTranscriptLimit = 500

type ticketRunResponse struct {
	ID                 string                              `json:"id"`
	TicketID           string                              `json:"ticket_id"`
	AttemptNumber      int                                 `json:"attempt_number"`
	AgentID            string                              `json:"agent_id"`
	AgentName          string                              `json:"agent_name"`
	Provider           string                              `json:"provider"`
	Status             string                              `json:"status"`
	CurrentStepStatus  *string                             `json:"current_step_status,omitempty"`
	CurrentStepSummary *string                             `json:"current_step_summary,omitempty"`
	CreatedAt          string                              `json:"created_at"`
	RuntimeStartedAt   *string                             `json:"runtime_started_at,omitempty"`
	LastHeartbeatAt    *string                             `json:"last_heartbeat_at,omitempty"`
	TerminalAt         *string                             `json:"terminal_at,omitempty"`
	CompletedAt        *string                             `json:"completed_at,omitempty"`
	LastError          *string                             `json:"last_error,omitempty"`
	CompletionSummary  *ticketRunCompletionSummaryResponse `json:"completion_summary,omitempty"`
}

type ticketRunCompletionSummaryResponse struct {
	Status      string         `json:"status"`
	Markdown    *string        `json:"markdown,omitempty"`
	JSON        map[string]any `json:"json,omitempty"`
	GeneratedAt *string        `json:"generated_at,omitempty"`
	Error       *string        `json:"error,omitempty"`
}

type ticketRunTraceEntryResponse struct {
	ID         string         `json:"id"`
	TicketID   string         `json:"ticket_id"`
	AgentRunID string         `json:"agent_run_id"`
	Sequence   int64          `json:"sequence"`
	Provider   string         `json:"provider"`
	Kind       string         `json:"kind"`
	Stream     string         `json:"stream"`
	Output     string         `json:"output"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  string         `json:"created_at"`
}

type ticketRunStepEntryResponse struct {
	ID                 string  `json:"id"`
	TicketID           string  `json:"ticket_id"`
	AgentRunID         string  `json:"agent_run_id"`
	StepStatus         string  `json:"step_status"`
	Summary            string  `json:"summary"`
	SourceTraceEventID *string `json:"source_trace_event_id,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

type ticketRunLifecycleEventResponse struct {
	EventType string `json:"event_type"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type ticketRunCatalog struct {
	runs      []domain.AgentRun
	attempts  map[uuid.UUID]int
	agents    map[uuid.UUID]domain.Agent
	providers map[uuid.UUID]domain.AgentProvider
}

type ticketRunActivityEnvelope struct {
	Event struct {
		ProjectID string         `json:"project_id"`
		TicketID  *string        `json:"ticket_id,omitempty"`
		EventType string         `json:"event_type"`
		Message   string         `json:"message"`
		Metadata  map[string]any `json:"metadata"`
		CreatedAt string         `json:"created_at"`
	} `json:"event"`
}

type ticketRunTraceEnvelope struct {
	Entry struct {
		ID         string         `json:"id"`
		ProjectID  string         `json:"project_id"`
		TicketID   string         `json:"ticket_id"`
		AgentRunID string         `json:"agent_run_id"`
		Sequence   int64          `json:"sequence"`
		Provider   string         `json:"provider"`
		Kind       string         `json:"kind"`
		Stream     string         `json:"stream"`
		Output     string         `json:"output"`
		Payload    map[string]any `json:"payload"`
		CreatedAt  string         `json:"created_at"`
	} `json:"entry"`
}

type ticketRunStepEnvelope struct {
	Entry struct {
		ID                 string  `json:"id"`
		ProjectID          string  `json:"project_id"`
		TicketID           string  `json:"ticket_id"`
		AgentRunID         string  `json:"agent_run_id"`
		StepStatus         string  `json:"step_status"`
		Summary            string  `json:"summary"`
		SourceTraceEventID *string `json:"source_trace_event_id,omitempty"`
		CreatedAt          string  `json:"created_at"`
	} `json:"entry"`
}

type ticketRunSummaryEnvelope struct {
	ProjectID         string                              `json:"project_id"`
	TicketID          string                              `json:"ticket_id"`
	RunID             string                              `json:"run_id"`
	CompletionSummary *ticketRunCompletionSummaryResponse `json:"completion_summary,omitempty"`
}

func (s *Server) handleListTicketRuns(c echo.Context) error {
	if s.ticketService == nil || s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "ticket run service unavailable")
	}

	projectID, ticketID, err := parseTicketRunPathParams(c)
	if err != nil {
		return err
	}
	if err := s.ensureTicketBelongsToProject(c.Request().Context(), projectID, ticketID); err != nil {
		return writeTicketError(c, err)
	}

	catalog, err := s.loadTicketRunCatalog(c.Request().Context(), projectID, ticketID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"runs": mapTicketRunResponses(catalog),
	})
}

func (s *Server) handleGetTicketRun(c echo.Context) error {
	if s.ticketService == nil || s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "ticket run service unavailable")
	}

	projectID, ticketID, err := parseTicketRunPathParams(c)
	if err != nil {
		return err
	}
	runID, err := parseUUIDPathParam(c, "runId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_RUN_ID", err.Error())
	}
	if err := s.ensureTicketBelongsToProject(c.Request().Context(), projectID, ticketID); err != nil {
		return writeTicketError(c, err)
	}

	catalog, err := s.loadTicketRunCatalog(c.Request().Context(), projectID, ticketID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	runItem, ok := findTicketRun(catalog.runs, runID)
	if !ok {
		return writeCatalogError(c, catalogservice.ErrNotFound)
	}

	traceEntries, err := s.catalog.ListAgentRunTraceEntries(c.Request().Context(), domain.ListAgentRunTraceEntries{
		ProjectID:  projectID,
		AgentRunID: runID,
		Limit:      ticketRunTranscriptLimit,
	})
	if err != nil {
		return writeCatalogError(c, err)
	}
	stepEntries, err := s.catalog.ListAgentRunStepEntries(c.Request().Context(), domain.ListAgentRunStepEntries{
		ProjectID:  projectID,
		AgentRunID: runID,
		Limit:      ticketRunTranscriptLimit,
	})
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"run":           mapTicketRunResponse(runItem, catalog),
		"trace_entries": mapTicketRunTraceEntryResponses(traceEntries),
		"step_entries":  mapTicketRunStepEntryResponses(stepEntries),
	})
}

func (s *Server) handleStreamTicketRuns(c echo.Context) error {
	if s.ticketService == nil || s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "ticket run service unavailable")
	}

	projectID, ticketID, err := parseTicketRunPathParams(c)
	if err != nil {
		return err
	}
	if err := s.ensureTicketBelongsToProject(c.Request().Context(), projectID, ticketID); err != nil {
		return writeTicketError(c, err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable ticket run sse write deadline: %w", err)
	}

	activityStream, err := s.sseHub.Register(streamCtx, ticketRunActivityStreamTopic)
	if err != nil {
		return fmt.Errorf("register ticket run activity stream: %w", err)
	}
	traceStream, err := s.sseHub.Register(streamCtx, agentTraceStreamTopic)
	if err != nil {
		return fmt.Errorf("register ticket run trace stream: %w", err)
	}
	stepStream, err := s.sseHub.Register(streamCtx, agentStepStreamTopic)
	if err != nil {
		return fmt.Errorf("register ticket run step stream: %w", err)
	}
	summaryStream, err := s.sseHub.Register(streamCtx, ticketRunStreamTopic)
	if err != nil {
		return fmt.Errorf("register ticket run summary stream: %w", err)
	}

	response := c.Response()
	header := response.Header()
	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	if err := writeSSEKeepaliveComment(response); err != nil {
		return err
	}

	heartbeat := time.NewTicker(sseKeepaliveInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-summaryStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := s.buildTicketRunSummaryStreamEvent(
				streamCtx,
				projectID,
				ticketID,
				event,
			)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run summary stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
			continue
		default:
		}

		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-activityStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := s.buildTicketRunLifecycleStreamEvent(
				streamCtx,
				projectID,
				ticketID,
				event,
			)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run lifecycle stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
			continue
		default:
		}

		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-activityStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := s.buildTicketRunLifecycleStreamEvent(
				streamCtx,
				projectID,
				ticketID,
				event,
			)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run lifecycle stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
		case event, ok := <-traceStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := buildTicketRunTraceStreamEvent(projectID, ticketID, event)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run trace stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
		case event, ok := <-stepStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := buildTicketRunStepStreamEvent(projectID, ticketID, event)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run step stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
		case event, ok := <-summaryStream:
			if !ok {
				return nil
			}
			streamEvent, matched, buildErr := s.buildTicketRunSummaryStreamEvent(
				streamCtx,
				projectID,
				ticketID,
				event,
			)
			if buildErr != nil {
				s.logger.Warn("skip malformed ticket run summary stream event", "error", buildErr)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, streamEvent); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEKeepaliveComment(response); err != nil {
				return err
			}
		}
	}
}

func parseTicketRunPathParams(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	projectID, err := parseProjectID(c)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	ticketID, err := parseTicketID(c)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	return projectID, ticketID, nil
}

func (s *Server) ensureTicketBelongsToProject(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) error {
	item, err := s.ticketService.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	if item.ProjectID != projectID {
		return ticketservice.ErrTicketNotFound
	}
	return nil
}

func (s *Server) loadTicketRunCatalog(ctx context.Context, projectID uuid.UUID, ticketID uuid.UUID) (ticketRunCatalog, error) {
	runs, err := s.catalog.ListAgentRuns(ctx, projectID)
	if err != nil {
		return ticketRunCatalog{}, err
	}

	filteredRuns := make([]domain.AgentRun, 0)
	for _, item := range runs {
		if item.TicketID == ticketID {
			filteredRuns = append(filteredRuns, item)
		}
	}
	sort.Slice(filteredRuns, func(i, j int) bool {
		if filteredRuns[i].CreatedAt.Equal(filteredRuns[j].CreatedAt) {
			return filteredRuns[i].ID.String() < filteredRuns[j].ID.String()
		}
		return filteredRuns[i].CreatedAt.Before(filteredRuns[j].CreatedAt)
	})

	agents, err := s.catalog.ListAgents(ctx, projectID)
	if err != nil {
		return ticketRunCatalog{}, err
	}
	agentIndex := make(map[uuid.UUID]domain.Agent, len(agents))
	for _, item := range agents {
		agentIndex[item.ID] = item
	}

	providerIndex := map[uuid.UUID]domain.AgentProvider{}
	for _, item := range filteredRuns {
		if _, ok := providerIndex[item.ProviderID]; ok {
			continue
		}
		providerItem, providerErr := s.catalog.GetAgentProvider(ctx, item.ProviderID)
		if providerErr != nil {
			return ticketRunCatalog{}, providerErr
		}
		providerIndex[item.ProviderID] = providerItem
	}

	attempts := make(map[uuid.UUID]int, len(filteredRuns))
	for idx, item := range filteredRuns {
		attempts[item.ID] = idx + 1
	}

	return ticketRunCatalog{
		runs:      filteredRuns,
		attempts:  attempts,
		agents:    agentIndex,
		providers: providerIndex,
	}, nil
}

func findTicketRun(runs []domain.AgentRun, runID uuid.UUID) (domain.AgentRun, bool) {
	for _, item := range runs {
		if item.ID == runID {
			return item, true
		}
	}
	return domain.AgentRun{}, false
}

func mapTicketRunResponses(catalog ticketRunCatalog) []ticketRunResponse {
	runs := append([]domain.AgentRun{}, catalog.runs...)
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CreatedAt.Equal(runs[j].CreatedAt) {
			return runs[i].ID.String() > runs[j].ID.String()
		}
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	response := make([]ticketRunResponse, 0, len(runs))
	for _, item := range runs {
		response = append(response, mapTicketRunResponse(item, catalog))
	}
	return response
}

func mapTicketRunResponse(item domain.AgentRun, catalog ticketRunCatalog) ticketRunResponse {
	agentName := "Unknown agent"
	if agentItem, ok := catalog.agents[item.AgentID]; ok && strings.TrimSpace(agentItem.Name) != "" {
		agentName = agentItem.Name
	}

	providerName := "unknown"
	if providerItem, ok := catalog.providers[item.ProviderID]; ok && strings.TrimSpace(providerItem.Name) != "" {
		providerName = providerItem.Name
	}

	var completedAt *string
	if item.Status == domain.AgentRunStatusCompleted {
		completedAt = timeToStringPointer(item.TerminalAt)
	}

	return ticketRunResponse{
		ID:                 item.ID.String(),
		TicketID:           item.TicketID.String(),
		AttemptNumber:      catalog.attempts[item.ID],
		AgentID:            item.AgentID.String(),
		AgentName:          agentName,
		Provider:           providerName,
		Status:             mapTicketRunStatus(item.Status),
		CurrentStepStatus:  copyStringPointer(item.CurrentStepStatus),
		CurrentStepSummary: copyStringPointer(item.CurrentStepSummary),
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
		RuntimeStartedAt:   timeToStringPointer(item.RuntimeStartedAt),
		LastHeartbeatAt:    timeToStringPointer(item.LastHeartbeatAt),
		TerminalAt:         timeToStringPointer(item.TerminalAt),
		CompletedAt:        completedAt,
		LastError:          optionalTrimmedString(item.LastError),
		CompletionSummary:  mapTicketRunCompletionSummaryResponse(item),
	}
}

func mapTicketRunCompletionSummaryResponse(item domain.AgentRun) *ticketRunCompletionSummaryResponse {
	if item.CompletionSummaryStatus == nil {
		return nil
	}

	return &ticketRunCompletionSummaryResponse{
		Status:      item.CompletionSummaryStatus.String(),
		Markdown:    copyStringPointer(item.CompletionSummaryMarkdown),
		JSON:        cloneMap(item.CompletionSummaryJSON),
		GeneratedAt: timeToStringPointer(item.CompletionSummaryGeneratedAt),
		Error:       copyStringPointer(item.CompletionSummaryError),
	}
}

func mapTicketRunStatus(status domain.AgentRunStatus) string {
	switch status {
	case domain.AgentRunStatusErrored:
		return "failed"
	case domain.AgentRunStatusTerminated:
		return "ended"
	default:
		return status.String()
	}
}

func mapTicketRunTraceEntryResponses(items []domain.AgentTraceEntry) []ticketRunTraceEntryResponse {
	response := make([]ticketRunTraceEntryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, ticketRunTraceEntryResponse{
			ID:         item.ID.String(),
			TicketID:   item.TicketID.String(),
			AgentRunID: item.AgentRunID.String(),
			Sequence:   item.Sequence,
			Provider:   item.Provider,
			Kind:       item.Kind,
			Stream:     item.Stream,
			Output:     item.Output,
			Payload:    cloneMap(item.Payload),
			CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func mapTicketRunStepEntryResponses(items []domain.AgentStepEntry) []ticketRunStepEntryResponse {
	response := make([]ticketRunStepEntryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, ticketRunStepEntryResponse{
			ID:                 item.ID.String(),
			TicketID:           item.TicketID.String(),
			AgentRunID:         item.AgentRunID.String(),
			StepStatus:         item.StepStatus,
			Summary:            item.Summary,
			SourceTraceEventID: uuidToStringPointer(item.SourceTraceEventID),
			CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func (s *Server) buildTicketRunLifecycleStreamEvent(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	if !strings.HasPrefix(event.Type.String(), "agent.") {
		return provider.Event{}, false, nil
	}

	var payload ticketRunActivityEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode ticket run lifecycle payload: %w", err)
	}
	if payload.Event.ProjectID != projectID.String() || payload.Event.TicketID == nil || *payload.Event.TicketID != ticketID.String() {
		return provider.Event{}, false, nil
	}
	if !isTicketRunLifecycleEventType(payload.Event.EventType) {
		return provider.Event{}, false, nil
	}

	runIDText, _ := payload.Event.Metadata["run_id"].(string)
	runIDText = strings.TrimSpace(runIDText)
	if runIDText == "" {
		return provider.Event{}, false, nil
	}
	runID, err := uuid.Parse(runIDText)
	if err != nil {
		return provider.Event{}, false, nil
	}

	catalog, err := s.loadTicketRunCatalog(ctx, projectID, ticketID)
	if err != nil {
		return provider.Event{}, false, err
	}
	runItem, ok := findTicketRun(catalog.runs, runID)
	if !ok {
		return provider.Event{}, false, nil
	}

	streamEvent, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunLifecycleStreamType,
		map[string]any{
			"run": mapTicketRunResponse(runItem, catalog),
			"lifecycle": ticketRunLifecycleEventResponse{
				EventType: payload.Event.EventType,
				Message:   payload.Event.Message,
				CreatedAt: payload.Event.CreatedAt,
			},
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct ticket run lifecycle stream event: %w", err)
	}
	return streamEvent, true, nil
}

func (s *Server) buildProjectTicketRunLifecycleStreamEvent(
	ctx context.Context,
	projectID uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	var payload ticketRunActivityEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode project ticket run lifecycle payload: %w", err)
	}
	if payload.Event.ProjectID != projectID.String() || payload.Event.TicketID == nil {
		return provider.Event{}, false, nil
	}

	ticketID, err := uuid.Parse(strings.TrimSpace(*payload.Event.TicketID))
	if err != nil {
		return provider.Event{}, false, nil
	}

	return s.buildTicketRunLifecycleStreamEvent(ctx, projectID, ticketID, event)
}

func (s *Server) buildTicketRunSummaryStreamEvent(
	ctx context.Context,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	if event.Topic != ticketRunStreamTopic || event.Type != ticketRunSummaryStreamType {
		return provider.Event{}, false, nil
	}

	var payload ticketRunSummaryEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode ticket run summary payload: %w", err)
	}
	if payload.ProjectID != projectID.String() || payload.TicketID != ticketID.String() {
		return provider.Event{}, false, nil
	}
	runID, err := uuid.Parse(strings.TrimSpace(payload.RunID))
	if err != nil {
		return provider.Event{}, false, nil
	}

	catalog, err := s.loadTicketRunCatalog(ctx, projectID, ticketID)
	if err != nil {
		return provider.Event{}, false, err
	}
	runItem, ok := findTicketRun(catalog.runs, runID)
	if !ok {
		return provider.Event{}, false, nil
	}

	streamEvent, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunSummaryStreamType,
		map[string]any{
			"project_id":         payload.ProjectID,
			"ticket_id":          payload.TicketID,
			"run_id":             payload.RunID,
			"run":                mapTicketRunResponse(runItem, catalog),
			"completion_summary": payload.CompletionSummary,
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct ticket run summary stream event: %w", err)
	}
	return streamEvent, true, nil
}

func (s *Server) buildProjectTicketRunSummaryStreamEvent(
	ctx context.Context,
	projectID uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	if event.Topic != ticketRunStreamTopic || event.Type != ticketRunSummaryStreamType {
		return provider.Event{}, false, nil
	}

	var payload ticketRunSummaryEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode project ticket run summary payload: %w", err)
	}
	if payload.ProjectID != projectID.String() {
		return provider.Event{}, false, nil
	}

	ticketID, err := uuid.Parse(strings.TrimSpace(payload.TicketID))
	if err != nil {
		return provider.Event{}, false, nil
	}

	return s.buildTicketRunSummaryStreamEvent(ctx, projectID, ticketID, event)
}

func buildTicketRunTraceStreamEvent(projectID uuid.UUID, ticketID uuid.UUID, event provider.Event) (provider.Event, bool, error) {
	if event.Type.String() != domain.AgentOutputEventType && event.Type.String() != "agent.trace" {
		return provider.Event{}, false, nil
	}

	var payload ticketRunTraceEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode ticket run trace payload: %w", err)
	}
	if payload.Entry.ProjectID != projectID.String() || payload.Entry.TicketID != ticketID.String() {
		return provider.Event{}, false, nil
	}

	streamEvent, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunTraceStreamType,
		map[string]any{
			"entry": ticketRunTraceEntryResponse{
				ID:         payload.Entry.ID,
				TicketID:   payload.Entry.TicketID,
				AgentRunID: payload.Entry.AgentRunID,
				Sequence:   payload.Entry.Sequence,
				Provider:   payload.Entry.Provider,
				Kind:       payload.Entry.Kind,
				Stream:     payload.Entry.Stream,
				Output:     payload.Entry.Output,
				Payload:    cloneMap(payload.Entry.Payload),
				CreatedAt:  payload.Entry.CreatedAt,
			},
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct ticket run trace stream event: %w", err)
	}
	return streamEvent, true, nil
}

func buildProjectTicketRunTraceStreamEvent(projectID uuid.UUID, event provider.Event) (provider.Event, bool, error) {
	var payload ticketRunTraceEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode project ticket run trace payload: %w", err)
	}
	if payload.Entry.ProjectID != projectID.String() {
		return provider.Event{}, false, nil
	}

	ticketID, err := uuid.Parse(strings.TrimSpace(payload.Entry.TicketID))
	if err != nil {
		return provider.Event{}, false, nil
	}

	return buildTicketRunTraceStreamEvent(projectID, ticketID, event)
}

func buildTicketRunStepStreamEvent(projectID uuid.UUID, ticketID uuid.UUID, event provider.Event) (provider.Event, bool, error) {
	if event.Type.String() != domain.AgentStepEventType {
		return provider.Event{}, false, nil
	}

	var payload ticketRunStepEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode ticket run step payload: %w", err)
	}
	if payload.Entry.ProjectID != projectID.String() || payload.Entry.TicketID != ticketID.String() {
		return provider.Event{}, false, nil
	}

	streamEvent, err := provider.NewJSONEvent(
		ticketRunStreamTopic,
		ticketRunStepStreamType,
		map[string]any{
			"entry": ticketRunStepEntryResponse{
				ID:                 payload.Entry.ID,
				TicketID:           payload.Entry.TicketID,
				AgentRunID:         payload.Entry.AgentRunID,
				StepStatus:         payload.Entry.StepStatus,
				Summary:            payload.Entry.Summary,
				SourceTraceEventID: payload.Entry.SourceTraceEventID,
				CreatedAt:          payload.Entry.CreatedAt,
			},
		},
		event.PublishedAt,
	)
	if err != nil {
		return provider.Event{}, false, fmt.Errorf("construct ticket run step stream event: %w", err)
	}
	return streamEvent, true, nil
}

func buildProjectTicketRunStepStreamEvent(projectID uuid.UUID, event provider.Event) (provider.Event, bool, error) {
	var payload ticketRunStepEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode project ticket run step payload: %w", err)
	}
	if payload.Entry.ProjectID != projectID.String() {
		return provider.Event{}, false, nil
	}

	ticketID, err := uuid.Parse(strings.TrimSpace(payload.Entry.TicketID))
	if err != nil {
		return provider.Event{}, false, nil
	}

	return buildTicketRunStepStreamEvent(projectID, ticketID, event)
}

func isTicketRunLifecycleEventType(raw string) bool {
	switch activityevent.Type(raw) {
	case activityevent.TypeAgentClaimed,
		activityevent.TypeAgentLaunching,
		activityevent.TypeAgentReady,
		activityevent.TypeAgentExecuting,
		activityevent.TypeAgentPaused,
		activityevent.TypeAgentFailed,
		activityevent.TypeAgentCompleted,
		activityevent.TypeAgentTerminated:
		return true
	default:
		return false
	}
}
