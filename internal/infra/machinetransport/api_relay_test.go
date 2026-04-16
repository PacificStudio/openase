package machinetransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
)

func TestRuntimeProtocolClientHandlesAPIRelayRequest(t *testing.T) {
	t.Parallel()

	requestBody := []byte(`{"ok":true}`)
	client := newRuntimeProtocolClient(func(context.Context, runtimecontract.Envelope) error { return nil })
	client.apiRelay = func(_ context.Context, request runtimecontract.APIRelayRequest) (runtimecontract.APIRelayResponse, error) {
		if request.Method != http.MethodGet {
			t.Fatalf("api relay method = %q", request.Method)
		}
		if request.URL != "http://127.0.0.1:19836/api/v1/healthz" {
			t.Fatalf("api relay url = %q", request.URL)
		}
		return runtimecontract.APIRelayResponse{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       requestBody,
		}, nil
	}

	responses := make([]runtimecontract.Envelope, 0, 1)
	client.send = func(_ context.Context, envelope runtimecontract.Envelope) error {
		responses = append(responses, envelope)
		return nil
	}

	requestPayload, err := marshalRuntimePayload(runtimecontract.APIRelayRequest{Method: http.MethodGet, URL: "http://127.0.0.1:19836/api/v1/healthz"})
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}
	if err := client.HandleEnvelope(runtimecontract.Envelope{Version: runtimecontract.ProtocolVersion, Type: runtimecontract.MessageTypeRequest, RequestID: "req-1", Operation: runtimecontract.OperationAPIRelay, Payload: requestPayload}); err != nil {
		t.Fatalf("HandleEnvelope() error = %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("responses len = %d", len(responses))
	}
	if responses[0].Type != runtimecontract.MessageTypeResponse || responses[0].RequestID != "req-1" || responses[0].Operation != runtimecontract.OperationAPIRelay {
		t.Fatalf("unexpected response envelope: %+v", responses[0])
	}
	payload, err := runtimecontract.DecodePayload[runtimecontract.APIRelayResponse](responses[0])
	if err != nil {
		t.Fatalf("decode response payload: %v", err)
	}
	if payload.StatusCode != http.StatusOK || string(payload.Body) != string(requestBody) {
		t.Fatalf("unexpected api relay response: %+v", payload)
	}
}

func TestRuntimeLocalRelayServerBridgesConnectedSession(t *testing.T) {
	t.Setenv(machinechanneldomain.EnvMachineLocalRelayURL, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	manager := newRuntimeAPIRelayManager()
	var session *runtimeAPIRelaySession
	session = newRuntimeAPIRelaySession(func(_ context.Context, requestID string, request runtimecontract.APIRelayRequest) error {
		go func() {
			_ = session.HandleResponse(requestID, runtimecontract.APIRelayResponse{StatusCode: http.StatusOK, Status: "200 OK", Headers: map[string][]string{"Content-Type": {"application/json"}}, Body: []byte(`{"path":"` + request.URL + `"}`)})
		}()
		return nil
	})
	manager.SetSession(session)

	_, relayURL, err := startRuntimeLocalRelayServer(ctx, manager, address)
	if err != nil {
		t.Fatalf("startRuntimeLocalRelayServer() error = %v", err)
	}
	waitForRuntimeRelayHealth(t, relayURL)

	body, err := json.Marshal(machinechanneldomain.LocalRelayRequest{Method: http.MethodGet, URL: "http://127.0.0.1:19836/api/v1/platform/projects/current"})
	if err != nil {
		t.Fatalf("marshal local relay request: %v", err)
	}
	response, err := http.Post(relayURL+localRelayPath, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST relay: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	var payload machinechanneldomain.LocalRelayResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode relay response: %v", err)
	}
	if payload.StatusCode != http.StatusOK {
		t.Fatalf("relay status code = %d, want %d", payload.StatusCode, http.StatusOK)
	}
	if string(payload.Body) != `{"path":"http://127.0.0.1:19836/api/v1/platform/projects/current"}` {
		t.Fatalf("relay body = %s", string(payload.Body))
	}
}

func waitForRuntimeRelayHealth(t *testing.T, relayURL string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(relayURL + "/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for relay %s", relayURL)
}

func TestRuntimeLocalRelayManagerKeepsOlderSessionWhenNewerSessionCloses(t *testing.T) {
	t.Parallel()
	manager := newRuntimeAPIRelayManager()
	var first *runtimeAPIRelaySession
	first = newRuntimeAPIRelaySession(func(_ context.Context, requestID string, _ runtimecontract.APIRelayRequest) error {
		go func() {
			_ = first.HandleResponse(requestID, runtimecontract.APIRelayResponse{StatusCode: http.StatusOK, Status: "200 OK", Body: []byte("first")})
		}()
		return nil
	})
	firstID := manager.SetSession(first)
	var second *runtimeAPIRelaySession
	second = newRuntimeAPIRelaySession(func(_ context.Context, requestID string, _ runtimecontract.APIRelayRequest) error {
		go func() {
			_ = second.HandleResponse(requestID, runtimecontract.APIRelayResponse{StatusCode: http.StatusOK, Status: "200 OK", Body: []byte("second")})
		}()
		return nil
	})
	secondID := manager.SetSession(second)
	manager.ClearSession(secondID, second, context.Canceled)
	response, err := manager.RoundTrip(context.Background(), runtimecontract.APIRelayRequest{Method: http.MethodGet, URL: "http://example.com"})
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	if string(response.Body) != "first" {
		t.Fatalf("RoundTrip() body = %s", string(response.Body))
	}
	manager.ClearSession(firstID, first, context.Canceled)
}
