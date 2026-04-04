package provider

import (
	"context"
	"net/http"
)

// TraceProvider manages request/context propagation and span lifecycle.
type TraceProvider interface {
	ExtractHTTPContext(ctx context.Context, header http.Header) context.Context
	InjectHTTPHeaders(ctx context.Context, header http.Header)
	StartSpan(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, Span)
	Shutdown(ctx context.Context) error
}

// Span is the provider-facing span abstraction used across application layers.
type Span interface {
	End()
	RecordError(err error)
	SetAttributes(attrs ...SpanAttribute)
	SetStatus(code SpanStatusCode, description string)
	TraceID() string
	SpanID() string
}

type SpanStatusCode string

const (
	SpanStatusUnset SpanStatusCode = "unset"
	SpanStatusOK    SpanStatusCode = "ok"
	SpanStatusError SpanStatusCode = "error"
)

type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindServer   SpanKind = "server"
	SpanKindClient   SpanKind = "client"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

type SpanAttributeType uint8

const (
	SpanAttributeTypeString SpanAttributeType = iota + 1
	SpanAttributeTypeInt64
	SpanAttributeTypeBool
)

type SpanAttribute struct {
	Key         string
	Type        SpanAttributeType
	StringValue string
	Int64Value  int64
	BoolValue   bool
}

func StringAttribute(key string, value string) SpanAttribute {
	return SpanAttribute{
		Key:         key,
		Type:        SpanAttributeTypeString,
		StringValue: value,
	}
}

func Int64Attribute(key string, value int64) SpanAttribute {
	return SpanAttribute{
		Key:        key,
		Type:       SpanAttributeTypeInt64,
		Int64Value: value,
	}
}

func IntAttribute(key string, value int) SpanAttribute {
	return Int64Attribute(key, int64(value))
}

func BoolAttribute(key string, value bool) SpanAttribute {
	return SpanAttribute{
		Key:       key,
		Type:      SpanAttributeTypeBool,
		BoolValue: value,
	}
}

type SpanStartOption interface {
	applySpanStart(*spanStartConfig)
}

type spanStartConfig struct {
	kind       SpanKind
	attributes []SpanAttribute
}

type spanKindOption struct {
	kind SpanKind
}

func (o spanKindOption) applySpanStart(cfg *spanStartConfig) {
	cfg.kind = o.kind
}

func WithSpanKind(kind SpanKind) SpanStartOption {
	return spanKindOption{kind: kind}
}

type spanAttributesOption struct {
	attributes []SpanAttribute
}

func (o spanAttributesOption) applySpanStart(cfg *spanStartConfig) {
	cfg.attributes = append(cfg.attributes, o.attributes...)
}

func WithSpanAttributes(attrs ...SpanAttribute) SpanStartOption {
	return spanAttributesOption{attributes: attrs}
}

func ResolveSpanStartOptions(opts ...SpanStartOption) (SpanKind, []SpanAttribute) {
	cfg := spanStartConfig{kind: SpanKindInternal}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.applySpanStart(&cfg)
	}

	return cfg.kind, cfg.attributes
}
