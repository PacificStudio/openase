package controlplaneurl

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	EnvBaseURL                = "OPENASE_BASE_URL"
	EnvControlPlaneURL        = "OPENASE_CONTROL_PLANE_URL"
	EnvMachineControlPlaneURL = "OPENASE_MACHINE_CONTROL_PLANE_URL"
)

func ResolveControlPlaneURL(explicit string, host string, port int) (string, error) {
	for _, candidate := range []string{strings.TrimSpace(explicit), strings.TrimSpace(os.Getenv(EnvControlPlaneURL)), strings.TrimSpace(os.Getenv(EnvBaseURL))} {
		if candidate == "" {
			continue
		}
		parsed, err := url.ParseRequestURI(candidate)
		if err != nil {
			return "", fmt.Errorf("parse control-plane-url: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return "", fmt.Errorf("parse control-plane-url: url must include scheme and host")
		}
		return strings.TrimRight(candidate, "/"), nil
	}

	trimmedHost := strings.TrimSpace(host)
	switch trimmedHost {
	case "", "0.0.0.0", "::", "[::]":
		trimmedHost = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(trimmedHost, strconv.Itoa(port)), nil
}

func APIBaseURLFromControlPlaneURL(raw string, platform bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse control-plane url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("control-plane url must include scheme and host")
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported control-plane url scheme %q", parsed.Scheme)
	}

	path := strings.TrimRight(strings.TrimSpace(parsed.Path), "/")
	switch {
	case path == "":
		parsed.Path = "/api/v1"
	case strings.HasSuffix(path, "/api/v1/platform"):
		if platform {
			parsed.Path = path
		} else {
			parsed.Path = strings.TrimSuffix(path, "/platform")
		}
	case strings.HasSuffix(path, "/api/v1/machines/connect"):
		parsed.Path = strings.TrimSuffix(path, "/machines/connect")
	case strings.HasSuffix(path, "/api/v1"):
		parsed.Path = path
	default:
		parsed.Path = path + "/api/v1"
	}
	if platform && !strings.HasSuffix(parsed.Path, "/api/v1/platform") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/platform"
	}
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}
