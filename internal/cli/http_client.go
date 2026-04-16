package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
)

func defaultCLIHTTPDoer() platformHTTPDoer {
	relayURL := strings.TrimSpace(os.Getenv(domain.EnvMachineLocalRelayURL))
	if relayURL == "" {
		return http.DefaultClient
	}
	return machineRelayHTTPDoer{relayURL: relayURL, client: http.DefaultClient}
}

type machineRelayHTTPDoer struct {
	relayURL string
	client   platformHTTPDoer
}

func (d machineRelayHTTPDoer) Do(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, fmt.Errorf("relay request must not be nil")
	}
	var body []byte
	if request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body for relay: %w", err)
		}
		body = payload
		request.Body = io.NopCloser(bytes.NewReader(payload))
	}
	localRequestBody, err := json.Marshal(domain.LocalRelayRequest{
		Method:  request.Method,
		URL:     request.URL.String(),
		Headers: map[string][]string(request.Header.Clone()),
		Body:    body,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal relay payload: %w", err)
	}
	bridgeURL := strings.TrimRight(strings.TrimSpace(d.relayURL), "/") + "/__openase_cli_relay"
	localRequest, err := http.NewRequestWithContext(request.Context(), http.MethodPost, bridgeURL, bytes.NewReader(localRequestBody))
	if err != nil {
		return nil, fmt.Errorf("build local relay request: %w", err)
	}
	localRequest.Header.Set("Content-Type", "application/json")
	client := d.client
	if client == nil {
		client = defaultCLIHTTPDoer()
	}
	localResponse, err := client.Do(localRequest)
	if err != nil {
		return nil, fmt.Errorf("local relay transport: %w", err)
	}
	defer func() {
		_ = localResponse.Body.Close()
	}()
	payload, err := io.ReadAll(localResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("read local relay response: %w", err)
	}
	var relayResponse domain.LocalRelayResponse
	if err := json.Unmarshal(payload, &relayResponse); err != nil {
		return nil, fmt.Errorf("decode local relay response: %w", err)
	}
	if localResponse.StatusCode < http.StatusOK || localResponse.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(relayResponse.Error)
		if message == "" {
			message = strings.TrimSpace(string(payload))
		}
		return nil, fmt.Errorf("local relay transport: %s", message)
	}
	statusCode := relayResponse.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	status := strings.TrimSpace(relayResponse.Status)
	if status == "" {
		status = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
	}
	return &http.Response{
		StatusCode:    statusCode,
		Status:        status,
		Header:        http.Header(cloneRelayHeaderMap(relayResponse.Headers)),
		Body:          io.NopCloser(bytes.NewReader(relayResponse.Body)),
		ContentLength: int64(len(relayResponse.Body)),
		Request:       request,
	}, nil
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
		cloned[trimmedKey] = append([]string(nil), values...)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneRelayHeaderMap(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string][]string, len(headers))
	for key, values := range headers {
		copied := make([]string, 0, len(values))
		for _, value := range values {
			copied = append(copied, value)
		}
		cloned[key] = copied
	}
	return cloned
}
