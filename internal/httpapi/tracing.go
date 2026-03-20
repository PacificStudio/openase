package httpapi

import (
	"fmt"
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/labstack/echo/v4"
)

const (
	traceIDContextKey = "trace_id"
	spanIDContextKey  = "span_id"
	traceIDHeader     = "X-Trace-Id"
)

func (s *Server) traceRequest() echo.MiddlewareFunc {
	traceProvider := s.trace
	if traceProvider == nil {
		traceProvider = provider.NewNoopTraceProvider()
		s.trace = traceProvider
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()
			ctx := traceProvider.ExtractHTTPContext(request.Context(), request.Header)
			ctx, span := traceProvider.StartSpan(ctx, requestSpanName(c),
				provider.WithSpanKind(provider.SpanKindServer),
				provider.WithSpanAttributes(
					provider.StringAttribute("http.method", request.Method),
					provider.StringAttribute("http.route", requestPath(c)),
				),
			)
			defer span.End()

			request = request.WithContext(ctx)
			c.SetRequest(request)
			c.Set(traceIDContextKey, span.TraceID())
			c.Set(spanIDContextKey, span.SpanID())
			if traceID := span.TraceID(); traceID != "" {
				c.Response().Header().Set(traceIDHeader, traceID)
			}
			traceProvider.InjectHTTPHeaders(ctx, c.Response().Header())

			err := next(c)
			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			span.SetAttributes(
				provider.IntAttribute("http.status_code", statusCode),
				provider.StringAttribute("http.request_id", c.Response().Header().Get(echo.HeaderXRequestID)),
			)
			if err != nil {
				span.RecordError(err)
			}
			if statusCode >= http.StatusInternalServerError || err != nil {
				description := ""
				if err != nil {
					description = err.Error()
				}
				span.SetStatus(provider.SpanStatusError, description)
			} else {
				span.SetStatus(provider.SpanStatusOK, "")
			}

			return err
		}
	}
}

func requestSpanName(c echo.Context) string {
	return fmt.Sprintf("%s %s", c.Request().Method, requestPath(c))
}

func requestPath(c echo.Context) string {
	if path := c.Path(); path != "" {
		return path
	}

	return c.Request().URL.Path
}

func traceValue(c echo.Context, key string) string {
	value, ok := c.Get(key).(string)
	if !ok {
		return ""
	}

	return value
}
