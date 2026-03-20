package provider

import (
	"context"
	"net/http"
)

func NewNoopTraceProvider() TraceProvider {
	return noopTraceProvider{}
}

type noopTraceProvider struct{}

func (noopTraceProvider) ExtractHTTPContext(ctx context.Context, _ http.Header) context.Context {
	return ctx
}

func (noopTraceProvider) InjectHTTPHeaders(context.Context, http.Header) {}

func (noopTraceProvider) StartSpan(ctx context.Context, _ string, _ ...SpanStartOption) (context.Context, Span) {
	return ctx, noopSpan{}
}

func (noopTraceProvider) Shutdown(context.Context) error {
	return nil
}

type noopSpan struct{}

func (noopSpan) End() {}

func (noopSpan) RecordError(error) {}

func (noopSpan) SetAttributes(...SpanAttribute) {}

func (noopSpan) SetStatus(SpanStatusCode, string) {}

func (noopSpan) TraceID() string {
	return ""
}

func (noopSpan) SpanID() string {
	return ""
}
