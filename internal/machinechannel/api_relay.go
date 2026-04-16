package machinechannel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	"github.com/google/uuid"
)

const machineLocalRelayPath = "/__openase_cli_relay"

var errAPIRelayUnavailable = errors.New("machine api relay is not connected")

type apiRelayManager struct {
	mu      sync.RWMutex
	session *apiRelaySession
}

type apiRelaySession struct {
	sessionID string
	send      func(context.Context, domain.APIRelayRequest) error

	mu      sync.Mutex
	pending map[string]chan apiRelayResult
	closed  error
}

type apiRelayResult struct {
	response domain.APIRelayResponse
	err      error
}

func newAPIRelayManager() *apiRelayManager {
	return &apiRelayManager{}
}

func (m *apiRelayManager) SetSession(session *apiRelaySession) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.session != nil && m.session != session {
		m.session.close(errAPIRelayUnavailable)
	}
	m.session = session
	m.mu.Unlock()
}

func (m *apiRelayManager) ClearSession(session *apiRelaySession, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.session == session {
		m.session = nil
	}
	m.mu.Unlock()
	if session != nil {
		session.close(err)
	}
}

func (m *apiRelayManager) RoundTrip(ctx context.Context, request domain.APIRelayRequest) (domain.APIRelayResponse, error) {
	if m == nil {
		return domain.APIRelayResponse{}, errAPIRelayUnavailable
	}
	m.mu.RLock()
	session := m.session
	m.mu.RUnlock()
	if session == nil {
		return domain.APIRelayResponse{}, errAPIRelayUnavailable
	}
	return session.Do(ctx, request)
}

func newAPIRelaySession(sessionID string, send func(context.Context, domain.APIRelayRequest) error) *apiRelaySession {
	return &apiRelaySession{
		sessionID: strings.TrimSpace(sessionID),
		send:      send,
		pending:   map[string]chan apiRelayResult{},
	}
}

func (s *apiRelaySession) Do(ctx context.Context, request domain.APIRelayRequest) (domain.APIRelayResponse, error) {
	if s == nil || s.send == nil {
		return domain.APIRelayResponse{}, errAPIRelayUnavailable
	}
	requestID := strings.TrimSpace(request.RequestID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	request.RequestID = requestID
	responseCh := make(chan apiRelayResult, 1)

	s.mu.Lock()
	if s.closed != nil {
		closedErr := s.closed
		s.mu.Unlock()
		return domain.APIRelayResponse{}, closedErr
	}
	s.pending[requestID] = responseCh
	s.mu.Unlock()

	if err := s.send(ctx, request); err != nil {
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return domain.APIRelayResponse{}, err
	}

	select {
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return domain.APIRelayResponse{}, ctx.Err()
	case result := <-responseCh:
		if result.err != nil {
			return domain.APIRelayResponse{}, result.err
		}
		return result.response, nil
	}
}

func (s *apiRelaySession) HandleResponse(response domain.APIRelayResponse) error {
	if s == nil {
		return errAPIRelayUnavailable
	}
	requestID := strings.TrimSpace(response.RequestID)
	if requestID == "" {
		return fmt.Errorf("api relay response is missing request_id")
	}
	s.mu.Lock()
	responseCh, ok := s.pending[requestID]
	if ok {
		delete(s.pending, requestID)
	}
	closedErr := s.closed
	s.mu.Unlock()
	if !ok {
		if closedErr != nil {
			return closedErr
		}
		return nil
	}
	responseCh <- apiRelayResult{response: response}
	return nil
}

func (s *apiRelaySession) close(err error) {
	if s == nil {
		return
	}
	if err == nil {
		err = errAPIRelayUnavailable
	}
	s.mu.Lock()
	if s.closed != nil {
		s.mu.Unlock()
		return
	}
	s.closed = err
	pending := s.pending
	s.pending = map[string]chan apiRelayResult{}
	s.mu.Unlock()
	for _, responseCh := range pending {
		responseCh <- apiRelayResult{err: err}
	}
}

func startDaemonLocalRelayServer(ctx context.Context, relay *apiRelayManager, address string) (*http.Server, string, error) {
	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" {
		trimmedAddress = domain.DefaultMachineLocalRelayAddress
	}
	listener, err := net.Listen("tcp", trimmedAddress)
	if err != nil {
		return nil, "", fmt.Errorf("listen local machine relay on %s: %w", trimmedAddress, err)
	}
	relayURL := "http://" + listener.Addr().String()
	if envErr := os.Setenv(domain.EnvMachineLocalRelayURL, relayURL); envErr != nil {
		_ = listener.Close()
		return nil, "", fmt.Errorf("set %s: %w", domain.EnvMachineLocalRelayURL, envErr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(machineLocalRelayPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer func() {
			_ = r.Body.Close()
		}()
		var request domain.LocalRelayRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(domain.LocalRelayResponse{Error: "decode relay request: " + err.Error()})
			return
		}
		response, err := relay.RoundTrip(r.Context(), domain.APIRelayRequest{
			Method:  request.Method,
			URL:     request.URL,
			Headers: cloneRelayHeaders(request.Headers),
			Body:    append([]byte(nil), request.Body...),
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(domain.LocalRelayResponse{Error: err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(domain.LocalRelayResponse{
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Headers:    cloneRelayHeaders(response.Headers),
			Body:       append([]byte(nil), response.Body...),
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		_ = server.Serve(listener)
	}()
	return server, relayURL, nil
}

func cloneRelayHeaders(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		copied := make([]string, 0, len(values))
		for _, value := range values {
			copied = append(copied, value)
		}
		cloned[trimmedKey] = copied
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func localRelayRoundTrip(ctx context.Context, relayURL string, request domain.LocalRelayRequest) (*http.Response, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal local relay request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(strings.TrimSpace(relayURL), "/")+machineLocalRelayPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build local relay request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(httpRequest)
}
