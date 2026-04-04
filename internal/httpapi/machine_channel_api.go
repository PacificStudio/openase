package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	machineEventsTopic                  = provider.MustParseTopic("machine.events")
	machineChannelRegisteredEventType   = provider.MustParseEventType("machine.channel_registered")
	machineChannelDisconnectedEventType = provider.MustParseEventType("machine.channel_disconnected")
	machineChannelReconnectedEventType  = provider.MustParseEventType("machine.channel_reconnected")
	machineChannelAuthFailedEventType   = provider.MustParseEventType("machine.channel_auth_failed")
	machineConnectUpgrader              = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
)

func (s *Server) handleMachineConnect(c echo.Context) error {
	if s.machineChannel == nil || s.machineSessions == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "machine channel service unavailable")
	}
	conn, err := machineConnectUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		s.logger.Warn("upgrade machine websocket", "error", err)
		return nil
	}
	defer func() {
		_ = conn.Close()
	}()
	ctx := c.Request().Context()

	helloEnvelope, err := readMachineEnvelope(conn)
	if err != nil {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_hello", err)
		return nil
	}
	if helloEnvelope.Type != domain.MessageTypeHello {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_hello", domain.ErrUnexpectedMessage)
		return nil
	}
	if _, err := domain.DecodePayload[domain.Hello](helloEnvelope); err != nil {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_hello", err)
		return nil
	}

	authenticateEnvelope, err := readMachineEnvelope(conn)
	if err != nil {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_authenticate", err)
		return nil
	}
	if authenticateEnvelope.Type != domain.MessageTypeAuthenticate {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_authenticate", domain.ErrUnexpectedMessage)
		return nil
	}
	authenticatePayload, err := domain.DecodePayload[domain.Authenticate](authenticateEnvelope)
	if err != nil {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_authenticate", err)
		return nil
	}
	if strings.TrimSpace(authenticatePayload.TransportMode) != "ws_reverse" {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "mode_mismatch", domain.ErrConnectionMode)
		return nil
	}
	parsedMachineID, err := uuid.Parse(strings.TrimSpace(authenticatePayload.MachineID))
	if err != nil {
		s.failMachineConnection(ctx, conn, uuid.Nil, "", "invalid_machine_id", err)
		return nil
	}
	claims, err := s.machineChannel.Authenticate(ctx, authenticatePayload.Token)
	if err != nil {
		s.failMachineConnection(ctx, conn, parsedMachineID, "", machineAuthFailureCode(err), err)
		return nil
	}
	if claims.MachineID != parsedMachineID {
		s.failMachineConnection(ctx, conn, parsedMachineID, "", "machine_id_mismatch", domain.ErrInvalidToken)
		return nil
	}

	connectedAt := time.Now().UTC()
	sessionID := uuid.NewString()
	registered, replaced := s.machineSessions.Register(parsedMachineID, sessionID, connectedAt, websocketSessionCloser{conn: conn})
	machineRecord, err := s.machineChannel.RecordConnectedSession(ctx, machinechannelservice.ConnectedSessionRecord{
		MachineID:        parsedMachineID,
		SessionID:        sessionID,
		ConnectedAt:      connectedAt,
		SystemInfo:       authenticatePayload.SystemInfo,
		ToolInventory:    authenticatePayload.ToolInventory,
		ResourceSnapshot: authenticatePayload.ResourceSnapshot,
	})
	if err != nil {
		_, _ = s.machineSessions.Remove(sessionID)
		s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "register_failed", err)
		return nil
	}

	if err := writeMachineEnvelope(conn, domain.MessageTypeRegistered, sessionID, domain.Registered{
		MachineID:                parsedMachineID.String(),
		SessionID:                sessionID,
		HeartbeatIntervalSeconds: int(machinechannelservice.DefaultHeartbeatInterval / time.Second),
		HeartbeatTimeoutSeconds:  int(machinechannelservice.DefaultHeartbeatTimeout / time.Second),
		ReplacedPreviousSession:  replaced != nil,
	}); err != nil {
		_, _ = s.machineSessions.Remove(sessionID)
		machineRecord, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
			MachineID:      parsedMachineID,
			SessionID:      sessionID,
			DisconnectedAt: time.Now().UTC(),
			Reason:         "registered_reply_failed",
		})
		s.recordMachineChannelActiveSessions()
		s.emitMachineChannelDisconnectActivityBestEffort(context.Background(), machineRecord.OrganizationID, parsedMachineID, sessionID, "registered_reply_failed")
		return nil
	}

	s.publishMachineChannelEvent(ctx, parsedMachineID, sessionID, registered.Replaced)
	s.recordMachineChannelMetric("registered")
	s.recordMachineChannelActiveSessions()
	s.emitMachineChannelActivityBestEffort(ctx, machineRecord.OrganizationID, parsedMachineID, sessionID, registered.Replaced)
	s.logger.Info(
		"machine reverse websocket registered",
		"machine_id", parsedMachineID.String(),
		"session_id", sessionID,
		"transport", "ws_reverse",
		"replaced_previous_session", registered.Replaced,
	)

	for {
		envelope, err := readMachineEnvelope(conn)
		if err != nil {
			break
		}
		switch envelope.Type {
		case domain.MessageTypeHeartbeat:
			heartbeatPayload, decodeErr := domain.DecodePayload[domain.Heartbeat](envelope)
			if decodeErr != nil {
				s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "invalid_heartbeat", decodeErr)
				return nil
			}
			heartbeatAt := time.Now().UTC()
			if _, ok := s.machineSessions.Heartbeat(sessionID, heartbeatAt); !ok {
				s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "session_replaced", domain.ErrSessionReplaced)
				return nil
			}
			if _, err := s.machineChannel.RecordHeartbeat(ctx, machinechannelservice.HeartbeatRecord{
				MachineID:        parsedMachineID,
				SessionID:        sessionID,
				HeartbeatAt:      heartbeatAt,
				SystemInfo:       heartbeatPayload.SystemInfo,
				ToolInventory:    heartbeatPayload.ToolInventory,
				ResourceSnapshot: heartbeatPayload.ResourceSnapshot,
			}); err != nil {
				s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "heartbeat_failed", err)
				return nil
			}
		case domain.MessageTypeGoodbye:
			goodbyePayload, _ := domain.DecodePayload[domain.Goodbye](envelope)
			_, _ = s.machineSessions.Remove(sessionID)
			machineRecord, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
				MachineID:      parsedMachineID,
				SessionID:      sessionID,
				DisconnectedAt: time.Now().UTC(),
				Reason:         strings.TrimSpace(goodbyePayload.Reason),
			})
			s.publishMachineChannelDisconnect(ctx, parsedMachineID, sessionID, "goodbye")
			s.recordMachineChannelActiveSessions()
			s.emitMachineChannelDisconnectActivityBestEffort(ctx, machineRecord.OrganizationID, parsedMachineID, sessionID, "goodbye")
			return nil
		default:
			s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "unexpected_message", domain.ErrUnexpectedMessage)
			return nil
		}
	}

	if _, ok := s.machineSessions.Remove(sessionID); !ok {
		return nil
	}
	machineRecord, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
		MachineID:      parsedMachineID,
		SessionID:      sessionID,
		DisconnectedAt: time.Now().UTC(),
		Reason:         "connection_closed",
	})
	s.publishMachineChannelDisconnect(ctx, parsedMachineID, sessionID, "connection_closed")
	s.recordMachineChannelActiveSessions()
	s.emitMachineChannelDisconnectActivityBestEffort(ctx, machineRecord.OrganizationID, parsedMachineID, sessionID, "connection_closed")
	return nil
}

func (s *Server) runMachineSessionExpiryLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for _, expired := range s.machineSessions.Expire(now.UTC()) {
				machineRecord, _ := s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
					MachineID:      expired.MachineID,
					SessionID:      expired.SessionID,
					DisconnectedAt: expired.DisconnectedAt,
					Reason:         "heartbeat_timeout",
				})
				s.publishMachineChannelDisconnect(context.Background(), expired.MachineID, expired.SessionID, "heartbeat_timeout")
				s.recordMachineChannelMetric("timeout")
				s.recordMachineChannelActiveSessions()
				s.emitMachineChannelDisconnectActivityBestEffort(context.Background(), machineRecord.OrganizationID, expired.MachineID, expired.SessionID, "heartbeat_timeout")
				s.logger.Warn(
					"machine reverse websocket expired",
					"machine_id", expired.MachineID.String(),
					"session_id", expired.SessionID,
					"transport", "ws_reverse",
				)
			}
		}
	}
}

func (s *Server) failMachineConnection(
	ctx context.Context,
	conn *websocket.Conn,
	machineID uuid.UUID,
	sessionID string,
	code string,
	err error,
) {
	_ = writeMachineEnvelope(conn, domain.MessageTypeError, sessionID, domain.ErrorPayload{
		Code:    code,
		Message: err.Error(),
	})
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, err.Error()),
		time.Now().Add(2*time.Second),
	)
	if machineID != uuid.Nil {
		payload := map[string]any{
			"machine_id":     machineID.String(),
			"session_id":     strings.TrimSpace(sessionID),
			"failure_code":   code,
			"transport_mode": "ws_reverse",
			"error":          err.Error(),
		}
		s.publishMachineTransportEvent(ctx, machineChannelAuthFailedEventType, payload)
		s.emitMachineChannelAuthFailedActivityBestEffort(ctx, machineID, sessionID, code, err)
	}
	s.recordMachineChannelMetric("auth_failed")
	s.logger.Warn(
		"machine reverse websocket handshake failed",
		"machine_id", machineID.String(),
		"session_id", sessionID,
		"transport", "ws_reverse",
		"failure_code", code,
		"error", err,
	)
}

func (s *Server) publishMachineChannelEvent(ctx context.Context, machineID uuid.UUID, sessionID string, replaced bool) {
	eventType := machineChannelRegisteredEventType
	if replaced {
		eventType = machineChannelReconnectedEventType
		s.recordMachineChannelMetric("reconnected")
		if s.metrics != nil {
			s.metrics.Counter("openase.machine_channel.websocket_reconnect_total", provider.Tags{
				"transport_mode": "ws_reverse",
			}).Add(1)
		}
	}
	s.publishMachineTransportEvent(ctx, eventType, map[string]any{
		"machine_id":     machineID.String(),
		"session_id":     sessionID,
		"transport_mode": "ws_reverse",
	})
}

func (s *Server) publishMachineChannelDisconnect(ctx context.Context, machineID uuid.UUID, sessionID string, reason string) {
	s.publishMachineTransportEvent(ctx, machineChannelDisconnectedEventType, map[string]any{
		"machine_id":     machineID.String(),
		"session_id":     sessionID,
		"transport_mode": "ws_reverse",
		"reason":         reason,
	})
	s.recordMachineChannelMetric("disconnected")
}

func (s *Server) publishMachineTransportEvent(ctx context.Context, eventType provider.EventType, payload map[string]any) {
	if s.events == nil {
		return
	}
	event, err := provider.NewJSONEvent(machineEventsTopic, eventType, payload, time.Now().UTC())
	if err == nil {
		_ = s.events.Publish(ctx, event)
	}
	_ = activityevent.TypeUnknown
}

func (s *Server) recordMachineChannelMetric(event string) {
	if s.metrics == nil {
		return
	}
	s.metrics.Counter("openase.machine_channel.events_total", provider.Tags{
		"event":          event,
		"transport_mode": "ws_reverse",
	}).Add(1)
}

func (s *Server) recordMachineChannelActiveSessions() {
	if s.metrics == nil || s.machineSessions == nil {
		return
	}
	s.metrics.Gauge("openase.machine_channel.active_sessions", provider.Tags{
		"transport_mode": "ws_reverse",
	}).Set(float64(s.machineSessions.Count()))
}

func (s *Server) emitMachineChannelActivityBestEffort(
	ctx context.Context,
	organizationID uuid.UUID,
	machineID uuid.UUID,
	sessionID string,
	replaced bool,
) {
	if organizationID == uuid.Nil {
		return
	}
	eventType := activityevent.TypeMachineConnected
	message := fmt.Sprintf("Machine %s connected over reverse websocket.", machineID)
	if replaced {
		eventType = activityevent.TypeMachineReconnected
		message = fmt.Sprintf("Machine %s reconnected over reverse websocket.", machineID)
	}
	if err := s.emitMachineTransportActivity(ctx, organizationID, machineID, eventType, message, map[string]any{
		"machine_id":      machineID.String(),
		"session_id":      strings.TrimSpace(sessionID),
		"transport_mode":  "ws_reverse",
		"connection_mode": "reverse_websocket",
	}); err != nil {
		s.logger.Warn("emit machine transport activity", "machine_id", machineID.String(), "session_id", sessionID, "event_type", eventType.String(), "error", err)
	}
}

func (s *Server) emitMachineChannelDisconnectActivityBestEffort(
	ctx context.Context,
	organizationID uuid.UUID,
	machineID uuid.UUID,
	sessionID string,
	reason string,
) {
	if organizationID == uuid.Nil {
		return
	}
	if err := s.emitMachineTransportActivity(ctx, organizationID, machineID, activityevent.TypeMachineDisconnected, fmt.Sprintf("Machine %s disconnected from reverse websocket.", machineID), map[string]any{
		"machine_id":      machineID.String(),
		"session_id":      strings.TrimSpace(sessionID),
		"transport_mode":  "ws_reverse",
		"connection_mode": "reverse_websocket",
		"reason":          strings.TrimSpace(reason),
	}); err != nil {
		s.logger.Warn("emit machine transport activity", "machine_id", machineID.String(), "session_id", sessionID, "event_type", activityevent.TypeMachineDisconnected.String(), "error", err)
	}
}

func (s *Server) emitMachineChannelAuthFailedActivityBestEffort(
	ctx context.Context,
	machineID uuid.UUID,
	sessionID string,
	code string,
	cause error,
) {
	if machineID == uuid.Nil || s == nil || s.catalog.MachineService == nil {
		return
	}
	machineItem, err := s.catalog.GetMachine(ctx, machineID)
	if err != nil {
		s.logger.Warn("load machine for auth failure activity", "machine_id", machineID.String(), "error", err)
		return
	}
	if err := s.emitMachineTransportActivity(ctx, machineItem.OrganizationID, machineID, activityevent.TypeMachineDaemonAuthFailed, fmt.Sprintf("Machine %s failed reverse websocket authentication.", machineItem.Name), map[string]any{
		"machine_id":      machineID.String(),
		"session_id":      strings.TrimSpace(sessionID),
		"transport_mode":  "ws_reverse",
		"connection_mode": "reverse_websocket",
		"failure_code":    strings.TrimSpace(code),
		"error":           machineTransportErrorString(cause),
	}); err != nil {
		s.logger.Warn("emit machine transport activity", "machine_id", machineID.String(), "session_id", sessionID, "event_type", activityevent.TypeMachineDaemonAuthFailed.String(), "error", err)
	}
}

func (s *Server) emitMachineTransportActivity(
	ctx context.Context,
	organizationID uuid.UUID,
	machineID uuid.UUID,
	eventType activityevent.Type,
	message string,
	metadata map[string]any,
) error {
	return s.emitMachineActivityForAffectedProjects(ctx, organizationID, machineID, func(projectID uuid.UUID) activitysvc.RecordInput {
		return activitysvc.RecordInput{
			ProjectID: projectID,
			EventType: eventType,
			Message:   message,
			Metadata:  metadata,
		}
	})
}

func machineTransportErrorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func readMachineEnvelope(conn *websocket.Conn) (domain.Envelope, error) {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return domain.Envelope{}, err
	}
	return domain.ParseEnvelope(payload)
}

func writeMachineEnvelope(conn *websocket.Conn, messageType domain.MessageType, sessionID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteJSON(domain.Envelope{
		Version:   domain.ProtocolVersion,
		Type:      messageType,
		SessionID: strings.TrimSpace(sessionID),
		Payload:   body,
	})
}

func machineAuthFailureCode(err error) string {
	switch {
	case errors.Is(err, domain.ErrTokenExpired):
		return "token_expired"
	case errors.Is(err, domain.ErrTokenRevoked):
		return "token_revoked"
	case errors.Is(err, domain.ErrConnectionMode):
		return "mode_mismatch"
	case errors.Is(err, domain.ErrMachineDisabled):
		return "machine_disabled"
	default:
		return "token_invalid"
	}
}

type websocketSessionCloser struct {
	conn *websocket.Conn
}

func (c websocketSessionCloser) Close(reason string) error {
	if c.conn == nil {
		return nil
	}
	return c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason),
		time.Now().Add(2*time.Second),
	)
}
