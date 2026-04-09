package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func requestExternalBaseURL(req *http.Request) string {
	scheme, host := forwardedSchemeAndHost(req)
	if scheme == "" || host == "" {
		return ""
	}
	return scheme + "://" + host
}

func forwardedSchemeAndHost(req *http.Request) (string, string) {
	if req == nil {
		return "", ""
	}
	if scheme, host := parseForwardedHeader(req.Header.Get("Forwarded")); scheme != "" && host != "" {
		return scheme, host
	}

	scheme := firstHeaderValue(req.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := firstHeaderValue(req.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(req.Host)
	}
	host = applyForwardedPort(host, firstHeaderValue(req.Header.Get("X-Forwarded-Port")))
	return strings.ToLower(scheme), host
}

func parseForwardedHeader(raw string) (string, string) {
	first := firstHeaderValue(raw)
	if first == "" {
		return "", ""
	}

	var scheme string
	var host string
	for _, part := range strings.Split(first, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "proto":
			scheme = strings.ToLower(value)
		case "host":
			host = value
		}
	}
	return scheme, host
}

func firstHeaderValue(raw string) string {
	first, _, _ := strings.Cut(raw, ",")
	return strings.TrimSpace(first)
}

func applyForwardedPort(host string, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" || port == "" {
		return host
	}
	parsed, err := url.Parse("//" + host)
	if err != nil || parsed.Port() != "" {
		return host
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return host
	}
	return net.JoinHostPort(hostname, port)
}
