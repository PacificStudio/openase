package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	controlplaneurl "github.com/BetterAndBetterII/openase/internal/controlplaneurl"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
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
	if remoteBinaryPath := strings.TrimSpace(result.RemoteBinaryPath); remoteBinaryPath != "" {
		envVars := domain.UpsertMachineEnvironmentValue(machine.EnvVars, "OPENASE_REAL_BIN", remoteBinaryPath)
		updateInput, err := parseMachinePatchRequest(machineID, machine, machinePatchRequest{EnvVars: &envVars})
		if err != nil {
			return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
		}
		updatedMachine, err := s.catalog.UpdateMachine(c.Request().Context(), updateInput)
		if err != nil {
			return writeCatalogError(c, err)
		}
		machine = updatedMachine
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
	if resolved, err := controlplaneurl.ResolveControlPlaneURL("", s.cfg.Host, s.cfg.Port); err == nil {
		configured := strings.TrimSpace(resolved)
		if configured != "" && !strings.Contains(configured, "://127.0.0.1") && !strings.Contains(configured, "://localhost") {
			return configured
		}
	}
	if external := strings.TrimSpace(requestExternalBaseURL(c.Request())); external != "" {
		return external
	}
	resolved, err := controlplaneurl.ResolveControlPlaneURL("", s.cfg.Host, s.cfg.Port)
	if err == nil {
		return resolved
	}
	host := strings.TrimSpace(s.cfg.Host)
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, s.cfg.Port)
}
