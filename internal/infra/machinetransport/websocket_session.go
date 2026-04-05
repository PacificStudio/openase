package machinetransport

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/gorilla/websocket"
)

const websocketWriteTimeout = 10 * time.Second

var websocketTransportComponent = logging.DeclareComponent("machine-transport-websocket")
var websocketTransportLogger = logging.WithComponent(nil, websocketTransportComponent)

type ProcessExitError struct {
	code int
}

func (e ProcessExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e ProcessExitError) ExitStatus() int {
	return e.code
}

func dialWebsocketRuntimeClient(
	ctx context.Context,
	machine domain.Machine,
) (*runtimeProtocolClient, func(error), error) {
	endpoint := strings.TrimSpace(pointerString(machine.AdvertisedEndpoint))
	if endpoint == "" {
		return nil, nil, fmt.Errorf("listener websocket endpoint is not configured for machine %s", machine.Name)
	}

	header := http.Header{}
	switch machine.ChannelCredential.Kind {
	case domain.MachineChannelCredentialKindNone, "":
	case domain.MachineChannelCredentialKindToken:
		token := strings.TrimSpace(pointerString(machine.ChannelCredential.TokenID))
		if token == "" {
			return nil, nil, fmt.Errorf("listener websocket token is not configured for machine %s", machine.Name)
		}
		header.Set("Authorization", "Bearer "+token)
	case domain.MachineChannelCredentialKindCertificate:
		return nil, nil, fmt.Errorf("listener websocket certificate credentials are not supported yet for machine %s", machine.Name)
	default:
		return nil, nil, fmt.Errorf("listener websocket credential kind %q is not supported", machine.ChannelCredential.Kind)
	}

	conn, response, err := (&websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}).DialContext(ctx, endpoint, header)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		classifiedErr := classifyWebsocketDialError(machine, endpoint, response, err)
		websocketTransportLogger.Warn("dial listener websocket failed", "machine_id", machine.ID.String(), "machine_name", machine.Name, "endpoint", endpoint, "error", classifiedErr)
		return nil, nil, classifiedErr
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}

	var writeMu sync.Mutex
	client := newRuntimeProtocolClient(func(ctx context.Context, envelope runtimecontract.Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
		return conn.WriteJSON(envelope)
	})
	go func() {
		defer client.Close(errors.New("listener websocket runtime closed"))
		for {
			var envelope runtimecontract.Envelope
			if err := conn.ReadJSON(&envelope); err != nil {
				return
			}
			if err := client.HandleEnvelope(envelope); err != nil {
				return
			}
		}
	}()

	closeFn := func(cause error) {
		client.Close(cause)
		_ = conn.Close()
	}
	if err := client.ensureHello(ctx); err != nil {
		closeFn(err)
		return nil, nil, err
	}
	websocketTransportLogger.Debug("dialed listener websocket", "machine_id", machine.ID.String(), "machine_name", machine.Name, "endpoint", endpoint)
	return client, closeFn, nil
}

type ListenerHandlerOptions struct {
	BearerToken string
}

func NewWebsocketListenerHandler(options ListenerHandlerOptions) http.Handler {
	return &websocketListenerHandler{
		bearerToken: strings.TrimSpace(options.BearerToken),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

type websocketListenerHandler struct {
	bearerToken string
	upgrader    websocket.Upgrader
}

func (h *websocketListenerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h == nil {
		http.Error(writer, "listener handler unavailable", http.StatusInternalServerError)
		return
	}
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.bearerToken != "" {
		token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
		if token != h.bearerToken {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := h.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	runWebsocketListenerSession(request.Context(), conn)
}

func runWebsocketListenerSession(parent context.Context, conn *websocket.Conn) {
	var writeMu sync.Mutex
	server := newRuntimeProtocolServer(func(ctx context.Context, envelope runtimecontract.Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
		return conn.WriteJSON(envelope)
	})
	defer server.Close()

	for {
		var envelope runtimecontract.Envelope
		if err := conn.ReadJSON(&envelope); err != nil {
			return
		}
		if err := server.HandleEnvelope(parent, envelope); err != nil {
			return
		}
	}
}

func classifyWebsocketDialError(
	machine domain.Machine,
	endpoint string,
	response *http.Response,
	err error,
) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("listener websocket authentication failed for machine %s at %s", machine.Name, endpoint)
		default:
			return fmt.Errorf("listener websocket handshake failed for machine %s at %s: %s", machine.Name, endpoint, response.Status)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("listener websocket DNS resolution failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return fmt.Errorf("listener websocket host verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var authorityErr x509.UnknownAuthorityError
	if errors.As(err, &authorityErr) {
		return fmt.Errorf("listener websocket TLS verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var opErr *net.OpError
	switch {
	case errors.As(err, &opErr):
		return fmt.Errorf("listener websocket endpoint unreachable for machine %s at %s: %w", machine.Name, endpoint, err)
	default:
		return fmt.Errorf("listener websocket dial failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}
}

func listenerShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-lc", command}
}
