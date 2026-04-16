package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/machinesetup"
	"github.com/labstack/echo/v4"
)

type sshBootstrapRequest struct {
	Topology            string `json:"topology,omitempty"`
	ListenerAddress     string `json:"listener_address,omitempty"`
	ListenerPath        string `json:"listener_path,omitempty"`
	ListenerBearerToken string `json:"listener_bearer_token,omitempty"`
	ControlPlaneURL     string `json:"control_plane_url,omitempty"`
	TokenTTLSeconds     int    `json:"token_ttl_seconds,omitempty"`
}

func (s *Server) sshBootstrapMachine(c echo.Context) error {
	if s.sshBootstrapper == nil {
		return c.JSON(http.StatusServiceUnavailable, errorResponse("ssh bootstrap is not configured on this server"))
	}

	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	var request sshBootstrapRequest
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	machine, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	controlPlaneURL := strings.TrimSpace(request.ControlPlaneURL)
	if controlPlaneURL == "" {
		controlPlaneURL = s.resolveControlPlaneURL(c)
	}

	result, err := s.sshBootstrapper.Bootstrap(c.Request().Context(), machine, machinesetup.BootstrapInput{
		Topology:            strings.TrimSpace(request.Topology),
		ListenerAddress:     strings.TrimSpace(request.ListenerAddress),
		ListenerPath:        strings.TrimSpace(request.ListenerPath),
		ListenerBearerToken: strings.TrimSpace(request.ListenerBearerToken),
		ControlPlaneURL:     controlPlaneURL,
		TokenTTLSeconds:     request.TokenTTLSeconds,
	})
	if err != nil {
		s.logger.Warn("ssh bootstrap failed",
			"machine_id", machineID.String(),
			"topology", strings.TrimSpace(request.Topology),
			"error", err,
		)
		return c.JSON(http.StatusBadGateway, errorResponse(fmt.Sprintf("ssh bootstrap failed: %v", err)))
	}

	s.logger.Info("ssh bootstrap completed",
		"machine_id", machineID.String(),
		"topology", result.Topology,
		"service_name", result.ServiceName,
		"service_status", result.ServiceStatus,
	)

	return c.JSON(http.StatusOK, map[string]any{"result": result})
}

// resolveControlPlaneURL derives the URL the remote machine-agent should dial
// back. Prefers the incoming request's Host so whatever proxy or domain the
// caller hit round-trips correctly, falling back to the configured host and
// port for bare localhost use.
func (s *Server) resolveControlPlaneURL(c echo.Context) string {
	if forwarded := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Host")); forwarded != "" {
		scheme := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Proto"))
		if scheme == "" {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s", scheme, forwarded)
	}
	if host := strings.TrimSpace(c.Request().Host); host != "" {
		scheme := "http"
		if c.IsTLS() {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	host := strings.TrimSpace(s.cfg.Host)
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, s.cfg.Port)
}
