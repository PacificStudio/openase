package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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
	sessionCtx, cleanup := s.longLivedRequestContext(ctx)
	defer cleanup()
	go func() {
		<-sessionCtx.Done()
		_ = websocketSessionCloser{conn: conn}.Close("server shutting down")
		_ = conn.Close()
	}()

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
	_, err = s.machineChannel.RecordConnectedSession(ctx, machinechannelservice.ConnectedSessionRecord{
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
		_, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
			MachineID:      parsedMachineID,
			SessionID:      sessionID,
			DisconnectedAt: time.Now().UTC(),
			Reason:         "registered_reply_failed",
		})
		return nil
	}

	s.publishMachineChannelEvent(ctx, parsedMachineID, sessionID, registered.Replaced)
	s.recordMachineChannelMetric("registered")
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
			_, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
				MachineID:      parsedMachineID,
				SessionID:      sessionID,
				DisconnectedAt: time.Now().UTC(),
				Reason:         strings.TrimSpace(goodbyePayload.Reason),
			})
			s.publishMachineChannelDisconnect(ctx, parsedMachineID, sessionID, "goodbye")
			return nil
		default:
			s.failMachineConnection(ctx, conn, parsedMachineID, sessionID, "unexpected_message", domain.ErrUnexpectedMessage)
			return nil
		}
	}

	_, _ = s.machineSessions.Remove(sessionID)
	_, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
		MachineID:      parsedMachineID,
		SessionID:      sessionID,
		DisconnectedAt: time.Now().UTC(),
		Reason:         "connection_closed",
	})
	s.publishMachineChannelDisconnect(ctx, parsedMachineID, sessionID, "connection_closed")
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
				_, _ = s.machineChannel.RecordDisconnectedSession(context.Background(), machinechannelservice.DisconnectedSessionRecord{
					MachineID:      expired.MachineID,
					SessionID:      expired.SessionID,
					DisconnectedAt: expired.DisconnectedAt,
					Reason:         "heartbeat_timeout",
				})
				s.publishMachineChannelDisconnect(context.Background(), expired.MachineID, expired.SessionID, "heartbeat_timeout")
				s.recordMachineChannelMetric("timeout")
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
		"event": event,
	}).Add(1)
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
