package httpapi

import (
	"net/http"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/labstack/echo/v4"
)

var (
	agentPlatformRouteMatrixOnce sync.Once
	agentPlatformRouteMatrix     map[string]struct{}
)

// HasAgentPlatformRoute reports whether the canonical human API operation has
// an /api/v1/platform counterpart.
func HasAgentPlatformRoute(method string, humanPath string) bool {
	signature := agentPlatformRouteSignature(method, humanPath)
	if signature == "" {
		return false
	}
	_, ok := getAgentPlatformRouteMatrix()[signature]
	return ok
}

func getAgentPlatformRouteMatrix() map[string]struct{} {
	agentPlatformRouteMatrixOnce.Do(func() {
		echoServer := echo.New()
		server := &Server{
			echo:          echoServer,
			agentPlatform: &agentplatform.Service{},
		}
		server.registerAgentPlatformRoutes(echoServer.Group("/api/v1/platform"))

		agentPlatformRouteMatrix = make(map[string]struct{})
		for _, route := range echoServer.Routes() {
			if route.Method == http.MethodHead || route.Method == http.MethodOptions {
				continue
			}
			signature := agentPlatformRouteSignature(route.Method, route.Path)
			if signature != "" {
				agentPlatformRouteMatrix[signature] = struct{}{}
			}
		}
	})
	return agentPlatformRouteMatrix
}

func agentPlatformRouteSignature(method string, path string) string {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	if normalizedMethod == "" {
		return ""
	}

	normalizedPath := normalizeAgentPlatformRoutePath(path)
	if normalizedPath == "" {
		return ""
	}
	return normalizedMethod + " " + normalizedPath
}

func normalizeAgentPlatformRoutePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimPrefix(trimmed, "/api/v1/platform")
	switch {
	case strings.HasPrefix(trimmed, "/api/v1/"):
	case strings.HasPrefix(trimmed, "/"):
		trimmed = "/api/v1" + trimmed
	case strings.HasPrefix(trimmed, "api/v1/"):
		trimmed = "/" + trimmed
	default:
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))
	for index := 0; index < len(trimmed); index++ {
		if trimmed[index] != '{' {
			builder.WriteByte(trimmed[index])
			continue
		}
		builder.WriteByte(':')
		index++
		for index < len(trimmed) && trimmed[index] != '}' {
			builder.WriteByte(trimmed[index])
			index++
		}
	}
	return builder.String()
}
