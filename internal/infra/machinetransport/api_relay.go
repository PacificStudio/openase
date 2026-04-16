package machinetransport

import (
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

	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/google/uuid"
)

const localRelayPath = "/__openase_cli_relay"

var _ = logging.DeclareComponent("machine-transport-runtime-api-relay")

var errRuntimeAPIRelayUnavailable = errors.New("runtime api relay is not connected")

type runtimeAPIRelayManager struct {
	mu      sync.RWMutex
	session *runtimeAPIRelaySession
}

type runtimeAPIRelaySession struct {
	mu      sync.Mutex
	send    func(context.Context, string, runtimecontract.APIRelayRequest) error
	pending map[string]chan runtimeAPIRelayResult
	closed  error
}

type runtimeAPIRelayResult struct {
	response runtimecontract.APIRelayResponse
	err      error
}

func newRuntimeAPIRelayManager() *runtimeAPIRelayManager { return &runtimeAPIRelayManager{} }

func (m *runtimeAPIRelayManager) SetSession(session *runtimeAPIRelaySession) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.session != nil && m.session != session {
		m.session.close(errRuntimeAPIRelayUnavailable)
	}
	m.session = session
	m.mu.Unlock()
}

func (m *runtimeAPIRelayManager) ClearSession(session *runtimeAPIRelaySession, err error) {
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

func (m *runtimeAPIRelayManager) RoundTrip(ctx context.Context, request runtimecontract.APIRelayRequest) (runtimecontract.APIRelayResponse, error) {
	if m == nil {
		return runtimecontract.APIRelayResponse{}, errRuntimeAPIRelayUnavailable
	}
	m.mu.RLock()
	session := m.session
	m.mu.RUnlock()
	if session == nil {
		return runtimecontract.APIRelayResponse{}, errRuntimeAPIRelayUnavailable
	}
	return session.Do(ctx, request)
}

func newRuntimeAPIRelaySession(send func(context.Context, string, runtimecontract.APIRelayRequest) error) *runtimeAPIRelaySession {
	return &runtimeAPIRelaySession{send: send, pending: map[string]chan runtimeAPIRelayResult{}}
}

func (s *runtimeAPIRelaySession) Do(ctx context.Context, request runtimecontract.APIRelayRequest) (runtimecontract.APIRelayResponse, error) {
	if s == nil || s.send == nil {
		return runtimecontract.APIRelayResponse{}, errRuntimeAPIRelayUnavailable
	}
	requestID := uuid.NewString()
	responseCh := make(chan runtimeAPIRelayResult, 1)
	s.mu.Lock()
	if s.closed != nil {
		err := s.closed
		s.mu.Unlock()
		return runtimecontract.APIRelayResponse{}, err
	}
	s.pending[requestID] = responseCh
	s.mu.Unlock()
	if err := s.send(ctx, requestID, request); err != nil {
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return runtimecontract.APIRelayResponse{}, err
	}
	select {
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return runtimecontract.APIRelayResponse{}, ctx.Err()
	case result := <-responseCh:
		if result.err != nil {
			return runtimecontract.APIRelayResponse{}, result.err
		}
		return result.response, nil
	}
}

func (s *runtimeAPIRelaySession) HandleResponse(requestID string, response runtimecontract.APIRelayResponse) error {
	if s == nil {
		return errRuntimeAPIRelayUnavailable
	}
	s.mu.Lock()
	ch, ok := s.pending[strings.TrimSpace(requestID)]
	if ok {
		delete(s.pending, strings.TrimSpace(requestID))
	}
	closed := s.closed
	s.mu.Unlock()
	if !ok {
		if closed != nil {
			return closed
		}
		return nil
	}
	ch <- runtimeAPIRelayResult{response: response}
	return nil
}

func (s *runtimeAPIRelaySession) close(err error) {
	if s == nil {
		return
	}
	if err == nil {
		err = errRuntimeAPIRelayUnavailable
	}
	s.mu.Lock()
	if s.closed != nil {
		s.mu.Unlock()
		return
	}
	s.closed = err
	pending := s.pending
	s.pending = map[string]chan runtimeAPIRelayResult{}
	s.mu.Unlock()
	for _, ch := range pending {
		ch <- runtimeAPIRelayResult{err: err}
	}
}

func startRuntimeLocalRelayServer(ctx context.Context, relay *runtimeAPIRelayManager, address string) (*http.Server, string, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		trimmed = machinechanneldomain.DefaultMachineLocalRelayAddress
	}
	listener, err := net.Listen("tcp", trimmed)
	if err != nil {
		return nil, "", fmt.Errorf("listen local runtime relay on %s: %w", trimmed, err)
	}
	relayURL := "http://" + listener.Addr().String()
	if err := os.Setenv(machinechanneldomain.EnvMachineLocalRelayURL, relayURL); err != nil {
		_ = listener.Close()
		return nil, "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(localRelayPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer func() { _ = r.Body.Close() }()
		var request machinechanneldomain.LocalRelayRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(machinechanneldomain.LocalRelayResponse{Error: err.Error()})
			return
		}
		response, err := relay.RoundTrip(r.Context(), runtimecontract.APIRelayRequest{Method: request.Method, URL: request.URL, Headers: cloneStringHeaders(request.Headers), Body: append([]byte(nil), request.Body...)})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(machinechanneldomain.LocalRelayResponse{Error: err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(machinechanneldomain.LocalRelayResponse{StatusCode: response.StatusCode, Status: response.Status, Headers: cloneStringHeaders(response.Headers), Body: append([]byte(nil), response.Body...)})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() { _ = server.Serve(listener) }()
	return server, relayURL, nil
}

func cloneStringHeaders(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		cloned[strings.TrimSpace(key)] = append([]string(nil), values...)
	}
	return cloned
}

type RuntimeLocalRelayManager = runtimeAPIRelayManager

func NewRuntimeLocalRelayManagerForCLI() *RuntimeLocalRelayManager {
	return newRuntimeAPIRelayManager()
}

func StartRuntimeLocalRelayServerForCLI(ctx context.Context, relay *RuntimeLocalRelayManager, address string) (*http.Server, string, error) {
	return startRuntimeLocalRelayServer(ctx, relay, address)
}
