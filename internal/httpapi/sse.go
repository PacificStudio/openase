package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const sseKeepaliveInterval = 5 * time.Second

var (
	ticketStreamTopic   = provider.MustParseTopic("ticket.events")
	agentStreamTopic    = provider.MustParseTopic("agent.events")
	hookStreamTopic     = provider.MustParseTopic("hook.events")
	activityStreamTopic = provider.MustParseTopic("activity.events")
	machineStreamTopic  = provider.MustParseTopic("machine.events")
	providerStreamTopic = provider.MustParseTopic("provider.events")

	projectDashboardStreamTopic      = provider.MustParseTopic("project.dashboard.events")
	projectDashboardRefreshEventType = provider.MustParseEventType("project.dashboard.refresh")
)

var projectDashboardRefreshDebounceInterval = time.Second

var projectPassiveStreamTopics = []provider.Topic{
	ticketStreamTopic,
	agentStreamTopic,
	hookStreamTopic,
	activityStreamTopic,
	agentTraceStreamTopic,
	agentStepStreamTopic,
	ticketRunStreamTopic,
}

type sseEnvelope struct {
	Topic       string          `json:"topic"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	PublishedAt time.Time       `json:"published_at"`
}

func (s *Server) handleEventStream(c echo.Context) error {
	topics, err := parseTopicQuery(c.QueryParams()["topic"])
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return s.handleEventStreamForTopics(c, topics...)
}

func (s *Server) handleProjectEventStream(c echo.Context) error {
	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable project event stream write deadline: %w", err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.sseHub.Register(streamCtx, projectPassiveStreamTopics...)
	if err != nil {
		return fmt.Errorf("register project event stream: %w", err)
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
	var dashboardFlushTimer *time.Timer
	var dashboardFlushC <-chan time.Time
	dashboardDirty := newProjectDashboardDirtySet()
	defer func() {
		if dashboardFlushTimer != nil {
			dashboardFlushTimer.Stop()
		}
	}()

	for {
		select {
		case <-streamCtx.Done():
			return nil
		case event, ok := <-stream:
			if !ok {
				return nil
			}

			streamEvents, buildErr := s.buildProjectStreamEvents(streamCtx, projectID, event)
			if buildErr != nil {
				s.logger.Warn(
					"skip malformed project event bus payload",
					"operation", "build_project_stream_events",
					"project_id", projectID,
					"topic", event.Topic.String(),
					"type", event.Type.String(),
					"payload_bytes", len(event.Payload),
					"error", buildErr,
				)
				continue
			}
			for _, streamEvent := range streamEvents {
				if err := writeSSEEvent(response, streamEvent); err != nil {
					return err
				}
			}
			if markProjectDashboardDirtySections(dashboardDirty, streamEvents) {
				switch {
				case dashboardFlushTimer == nil:
					dashboardFlushTimer = time.NewTimer(projectDashboardRefreshDebounceInterval)
					dashboardFlushC = dashboardFlushTimer.C
				case !dashboardFlushTimer.Stop():
					select {
					case <-dashboardFlushTimer.C:
					default:
					}
					dashboardFlushTimer.Reset(projectDashboardRefreshDebounceInterval)
				default:
					dashboardFlushTimer.Reset(projectDashboardRefreshDebounceInterval)
				}
			}
		case <-dashboardFlushC:
			if dashboardDirty.Empty() {
				dashboardFlushC = nil
				dashboardFlushTimer = nil
				continue
			}
			refreshEvent, buildErr := buildProjectDashboardRefreshEvent(projectID, dashboardDirty, time.Now().UTC())
			dashboardDirty.Clear()
			dashboardFlushC = nil
			dashboardFlushTimer = nil
			if buildErr != nil {
				s.logger.Warn(
					"skip malformed project dashboard refresh event",
					"operation", "build_project_dashboard_refresh_event",
					"project_id", projectID,
					"error", buildErr,
				)
				continue
			}
			if err := writeSSEEvent(response, refreshEvent); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEKeepaliveComment(response); err != nil {
				return err
			}
		}
	}
}

func (s *Server) handleMachineStream(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	return s.handleOrganizationScopedEventStream(c, machineStreamTopic, "machine", orgID)
}

func (s *Server) handleProviderStream(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	return s.handleOrganizationScopedEventStream(c, providerStreamTopic, "provider", orgID)
}

func (s *Server) handleEventStreamForTopics(c echo.Context, topics ...provider.Topic) error {
	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable sse write deadline: %w", err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.sseHub.Register(streamCtx, topics...)
	if err != nil {
		return fmt.Errorf("register sse connection: %w", err)
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
		case event, ok := <-stream:
			if !ok {
				return nil
			}
			if err := writeSSEEvent(response, event); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEKeepaliveComment(response); err != nil {
				return err
			}
		}
	}
}

func (s *Server) handleOrganizationScopedEventStream(
	c echo.Context,
	topic provider.Topic,
	streamName string,
	orgID uuid.UUID,
) error {
	if err := http.NewResponseController(c.Response().Writer).SetWriteDeadline(time.Time{}); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("disable sse write deadline: %w", err)
	}

	streamCtx, cancel := s.shutdownAwareContext(c.Request().Context())
	defer cancel()

	stream, err := s.sseHub.Register(streamCtx, topic)
	if err != nil {
		return fmt.Errorf("register %s stream: %w", streamName, err)
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
		case event, ok := <-stream:
			if !ok {
				return nil
			}

			scopedEvent, matched, err := buildOrganizationScopedStreamEvent(orgID, event)
			if err != nil {
				s.logger.Warn(
					"skip malformed organization-scoped stream event",
					"operation", "build_organization_scoped_stream_event",
					"organization_id", orgID,
					"topic", topic.String(),
					"type", event.Type.String(),
					"payload_bytes", len(event.Payload),
					"error", err,
				)
				continue
			}
			if !matched {
				continue
			}
			if err := writeSSEEvent(response, scopedEvent); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := writeSSEKeepaliveComment(response); err != nil {
				return err
			}
		}
	}
}

func parseTopicQuery(rawTopics []string) ([]provider.Topic, error) {
	if len(rawTopics) == 0 {
		return nil, fmt.Errorf("at least one topic query parameter is required")
	}

	topics := make([]provider.Topic, 0, len(rawTopics))
	for _, rawTopic := range rawTopics {
		topic, err := provider.ParseTopic(rawTopic)
		if err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}

	return topics, nil
}

func buildOrganizationScopedStreamEvent(
	orgID uuid.UUID,
	event provider.Event,
) (provider.Event, bool, error) {
	var payload struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return provider.Event{}, false, fmt.Errorf("decode organization-scoped payload: %w", err)
	}
	if payload.OrganizationID != orgID.String() {
		return provider.Event{}, false, nil
	}

	return event, true, nil
}

func (s *Server) buildProjectStreamEvents(
	ctx context.Context,
	projectID uuid.UUID,
	event provider.Event,
) ([]provider.Event, error) {
	switch event.Topic {
	case ticketStreamTopic:
		return buildProjectScopedPassthroughEvents(projectID, event, parseTicketStreamProjectID)
	case agentStreamTopic:
		return buildProjectScopedPassthroughEvents(projectID, event, parseAgentStreamProjectID)
	case hookStreamTopic:
		return buildProjectScopedPassthroughEvents(projectID, event, parseHookStreamProjectID)
	case activityStreamTopic:
		events, err := buildProjectScopedPassthroughEvents(projectID, event, parseActivityStreamProjectID)
		if err != nil || len(events) == 0 {
			return events, err
		}

		ticketRunEvent, matched, buildErr := s.buildProjectTicketRunLifecycleStreamEvent(ctx, projectID, event)
		if buildErr != nil {
			return nil, buildErr
		}
		if matched {
			events = append(events, ticketRunEvent)
		}
		return events, nil
	case agentTraceStreamTopic:
		ticketRunEvent, matched, err := buildProjectTicketRunTraceStreamEvent(projectID, event)
		if err != nil || !matched {
			return nil, err
		}
		return []provider.Event{ticketRunEvent}, nil
	case agentStepStreamTopic:
		ticketRunEvent, matched, err := buildProjectTicketRunStepStreamEvent(projectID, event)
		if err != nil || !matched {
			return nil, err
		}
		return []provider.Event{ticketRunEvent}, nil
	case ticketRunStreamTopic:
		ticketRunEvent, matched, err := s.buildProjectTicketRunSummaryStreamEvent(ctx, projectID, event)
		if err != nil || !matched {
			return nil, err
		}
		return []provider.Event{ticketRunEvent}, nil
	default:
		return nil, nil
	}
}

func buildProjectScopedPassthroughEvents(
	projectID uuid.UUID,
	event provider.Event,
	parseProjectID func(json.RawMessage) (string, error),
) ([]provider.Event, error) {
	rawProjectID, err := parseProjectID(event.Payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rawProjectID) != projectID.String() {
		return nil, nil
	}
	return []provider.Event{event}, nil
}

func parseTicketStreamProjectID(payload json.RawMessage) (string, error) {
	var envelope struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode ticket stream payload: %w", err)
	}
	return envelope.ProjectID, nil
}

func parseAgentStreamProjectID(payload json.RawMessage) (string, error) {
	var envelope struct {
		Agent struct {
			ProjectID string `json:"project_id"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode agent stream payload: %w", err)
	}
	return envelope.Agent.ProjectID, nil
}

func parseHookStreamProjectID(payload json.RawMessage) (string, error) {
	var envelope struct {
		ProjectID string `json:"project_id"`
		Hook      struct {
			ProjectID string `json:"project_id"`
		} `json:"hook"`
		Event struct {
			ProjectID string `json:"project_id"`
		} `json:"event"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode hook stream payload: %w", err)
	}

	switch {
	case strings.TrimSpace(envelope.ProjectID) != "":
		return envelope.ProjectID, nil
	case strings.TrimSpace(envelope.Hook.ProjectID) != "":
		return envelope.Hook.ProjectID, nil
	default:
		return envelope.Event.ProjectID, nil
	}
}

func parseActivityStreamProjectID(payload json.RawMessage) (string, error) {
	var envelope ticketRunActivityEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode activity stream payload: %w", err)
	}
	return envelope.Event.ProjectID, nil
}

func writeSSEKeepaliveComment(response *echo.Response) error {
	if _, err := fmt.Fprint(response, ": keepalive\n\n"); err != nil {
		return err
	}

	response.Flush()
	return nil
}

func writeSSEEvent(response *echo.Response, event provider.Event) error {
	payload, err := json.Marshal(sseEnvelope{
		Topic:       event.Topic.String(),
		Type:        event.Type.String(),
		Payload:     event.Payload,
		PublishedAt: event.PublishedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}

	if _, err := fmt.Fprintf(response, "event: %s\n", event.Type.String()); err != nil {
		return err
	}
	for _, line := range strings.Split(string(payload), "\n") {
		if _, err := fmt.Fprintf(response, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(response, "\n"); err != nil {
		return err
	}

	response.Flush()
	return nil
}

type projectDashboardDirtySet struct {
	project             bool
	agents              bool
	tickets             bool
	activity            bool
	hrAdvisor           bool
	organizationSummary bool
}

func newProjectDashboardDirtySet() *projectDashboardDirtySet {
	return &projectDashboardDirtySet{}
}

func (s *projectDashboardDirtySet) Empty() bool {
	return !s.project && !s.agents && !s.tickets && !s.activity && !s.hrAdvisor && !s.organizationSummary
}

func (s *projectDashboardDirtySet) Clear() {
	s.project = false
	s.agents = false
	s.tickets = false
	s.activity = false
	s.hrAdvisor = false
	s.organizationSummary = false
}

func (s *projectDashboardDirtySet) Sections() []string {
	sections := make([]string, 0, 6)
	if s.project {
		sections = append(sections, "project")
	}
	if s.agents {
		sections = append(sections, "agents")
	}
	if s.tickets {
		sections = append(sections, "tickets")
	}
	if s.activity {
		sections = append(sections, "activity")
	}
	if s.hrAdvisor {
		sections = append(sections, "hr_advisor")
	}
	if s.organizationSummary {
		sections = append(sections, "organization_summary")
	}
	return sections
}

func markProjectDashboardDirtySections(target *projectDashboardDirtySet, events []provider.Event) bool {
	if target == nil || len(events) == 0 {
		return false
	}

	before := target.Sections()
	for _, event := range events {
		switch event.Topic {
		case ticketStreamTopic:
			target.tickets = true
			target.hrAdvisor = true
			target.organizationSummary = true
		case agentStreamTopic:
			target.agents = true
			target.hrAdvisor = true
		case hookStreamTopic:
			target.activity = true
		case activityStreamTopic:
			target.activity = true
			target.hrAdvisor = true
			eventType, err := parseActivityStreamEventType(event.Payload)
			if err == nil && strings.HasPrefix(eventType, "project.") {
				target.project = true
			}
		}
	}

	return len(before) != len(target.Sections())
}

func parseActivityStreamEventType(payload json.RawMessage) (string, error) {
	var envelope ticketRunActivityEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", fmt.Errorf("decode activity stream payload for event type: %w", err)
	}
	return envelope.Event.EventType, nil
}

func buildProjectDashboardRefreshEvent(
	projectID uuid.UUID,
	dirty *projectDashboardDirtySet,
	publishedAt time.Time,
) (provider.Event, error) {
	if dirty == nil || dirty.Empty() {
		return provider.Event{}, fmt.Errorf("project dashboard refresh event requires dirty sections")
	}

	return provider.NewJSONEvent(
		projectDashboardStreamTopic,
		projectDashboardRefreshEventType,
		map[string]any{
			"project_id":     projectID.String(),
			"dirty_sections": dirty.Sections(),
		},
		publishedAt,
	)
}
