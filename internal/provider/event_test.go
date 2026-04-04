package provider

import (
	"testing"
	"time"
)

func TestParseTopicRejectsInvalidValue(t *testing.T) {
	if _, err := ParseTopic("Runtime Events"); err == nil {
		t.Fatal("expected invalid topic error")
	}
}

func TestNewJSONEventNormalizesPublishedAtToUTC(t *testing.T) {
	topic := MustParseTopic("ticket.events")
	eventType := MustParseEventType("ticket.updated")
	publishedAt := time.Date(2026, 3, 19, 11, 30, 0, 0, time.FixedZone("custom", 2*60*60))

	event, err := NewJSONEvent(topic, eventType, map[string]string{"status": "done"}, publishedAt)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	if event.PublishedAt.Location() != time.UTC {
		t.Fatalf("expected published_at in UTC, got %s", event.PublishedAt.Location())
	}
}
