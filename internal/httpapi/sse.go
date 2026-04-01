package httpapi

import (
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

const sseKeepaliveInterval = 25 * time.Second

var (
	ticketStreamTopic   = provider.MustParseTopic("ticket.events")
	agentStreamTopic    = provider.MustParseTopic("agent.events")
	hookStreamTopic     = provider.MustParseTopic("hook.events")
	activityStreamTopic = provider.MustParseTopic("activity.events")
	machineStreamTopic  = provider.MustParseTopic("machine.events")
	providerStreamTopic = provider.MustParseTopic("provider.events")
)

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

func (s *Server) handleTicketStream(c echo.Context) error {
	return s.handleEventStreamForTopics(c, ticketStreamTopic)
}

func (s *Server) handleAgentStream(c echo.Context) error {
	return s.handleEventStreamForTopics(c, agentStreamTopic)
}

func (s *Server) handleHookStream(c echo.Context) error {
	return s.handleEventStreamForTopics(c, hookStreamTopic)
}

func (s *Server) handleActivityStream(c echo.Context) error {
	return s.handleEventStreamForTopics(c, activityStreamTopic)
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

	stream, err := s.sseHub.Register(c.Request().Context(), topics...)
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
		case <-c.Request().Context().Done():
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

	stream, err := s.sseHub.Register(c.Request().Context(), topic)
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
		case <-c.Request().Context().Done():
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
