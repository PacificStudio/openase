package httpapi

import (
	"log/slog"
	"net/url"
	"sort"

	"github.com/labstack/echo/v4"
)

const requestLoggerContextKey = "httpapi.request_logger"

func (s *Server) injectRequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(requestLoggerContextKey, s.logger)
			return next(c)
		}
	}
}

func requestLogger(c echo.Context) *slog.Logger {
	if c != nil {
		if logger, ok := c.Get(requestLoggerContextKey).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return slog.Default()
}

func logAPIBoundaryError(c echo.Context, statusCode int, code string, message string) {
	if c == nil {
		return
	}

	attrs := []any{
		"operation", "api_boundary_error",
		"method", c.Request().Method,
		"route", requestRoute(c),
		"path", c.Request().URL.Path,
		"status_code", statusCode,
		"error_code", code,
		"error", message,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
		"trace_id", traceValue(c, traceIDContextKey),
		"span_id", traceValue(c, spanIDContextKey),
		"path_params", requestPathParams(c),
		"query_keys", requestQueryKeys(c.QueryParams()),
		"content_length", c.Request().ContentLength,
	}

	logger := requestLogger(c)
	if statusCode >= 500 {
		logger.Error("http api boundary error", attrs...)
		return
	}
	logger.Warn("http api boundary error", attrs...)
}

func requestRoute(c echo.Context) string {
	if route := c.Path(); route != "" {
		return route
	}
	return "unmatched"
}

func requestPathParams(c echo.Context) map[string]string {
	params := map[string]string{}
	for _, name := range c.ParamNames() {
		params[name] = c.Param(name)
	}
	return params
}

func requestQueryKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
