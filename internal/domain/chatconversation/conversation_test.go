package chatconversation

import (
	"strings"
	"testing"
)

func TestParseSource(t *testing.T) {
	t.Parallel()

	source, err := ParseSource("  project_sidebar  ")
	if err != nil {
		t.Fatalf("ParseSource() error = %v", err)
	}
	if source != SourceProjectSidebar {
		t.Fatalf("ParseSource() = %q, want %q", source, SourceProjectSidebar)
	}
}

func TestParseSourceRejectsUnsupportedValues(t *testing.T) {
	t.Parallel()

	if _, err := ParseSource("ticket_detail"); err == nil {
		t.Fatal("ParseSource() error = nil, want error")
	}
}

func TestDomainErrorsAreStableSentinels(t *testing.T) {
	t.Parallel()

	if ErrNotFound == nil || ErrConflict == nil || ErrInvalidInput == nil {
		t.Fatal("expected domain error sentinels to be initialized")
	}
	if ErrNotFound == ErrConflict || ErrConflict == ErrInvalidInput || ErrNotFound == ErrInvalidInput {
		t.Fatal("expected distinct domain error sentinels")
	}
}

func TestParseConversationTitleFromFirstUserMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    ConversationTitle
		wantErr bool
	}{
		{
			name: "chinese first sentence",
			raw:  "请帮我修复对话标题漂移的问题。后续 summary 不能再覆盖。",
			want: ConversationTitle("请帮我修复对话标题漂移的问题。"),
		},
		{
			name: "english first sentence",
			raw:  "Stabilize the conversation title now. The summary should stay secondary.",
			want: ConversationTitle("Stabilize the conversation title now."),
		},
		{
			name: "first non-empty line without punctuation",
			raw:  "\n\n   Keep the first line as the title   \nSecond line adds context",
			want: ConversationTitle("Keep the first line as the title"),
		},
		{
			name: "compresses whitespace and keeps title single-line",
			raw:  "   Tighten    the \n\n   conversation\t\t title semantics now!   More follows.",
			want: ConversationTitle("Tighten the conversation title semantics now!"),
		},
		{
			name: "truncates long titles server side",
			raw:  strings.Repeat("a", MaxConversationTitleRunes+10),
			want: ConversationTitle(strings.Repeat("a", MaxConversationTitleRunes)),
		},
		{
			name:    "rejects empty messages",
			raw:     " \n\t ",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseConversationTitleFromFirstUserMessage(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParseConversationTitleFromFirstUserMessage() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConversationTitleFromFirstUserMessage() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ParseConversationTitleFromFirstUserMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestConversationTitleHelpers(t *testing.T) {
	t.Parallel()

	if got := (ConversationTitle("  Stable title  ")).String(); got != "Stable title" {
		t.Fatalf("ConversationTitle.String() = %q, want %q", got, "Stable title")
	}
	if got := firstNonEmptyConversationTitleLine("\n  \n first line \nsecond line"); got != "first line" {
		t.Fatalf("firstNonEmptyConversationTitleLine() = %q, want %q", got, "first line")
	}
	if got := firstNonEmptyConversationTitleLine("\n\t "); got != "" {
		t.Fatalf("firstNonEmptyConversationTitleLine() = %q, want empty", got)
	}
	if got := firstConversationSentence(""); got != "" {
		t.Fatalf("firstConversationSentence(empty) = %q, want empty", got)
	}
	if got := firstConversationSentence("No punctuation here"); got != "" {
		t.Fatalf("firstConversationSentence(no punctuation) = %q, want empty", got)
	}
	if got := firstConversationSentence("Ends here! Keep going"); got != "Ends here!" {
		t.Fatalf("firstConversationSentence() = %q, want %q", got, "Ends here!")
	}
}
