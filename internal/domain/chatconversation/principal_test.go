package chatconversation

import "testing"

func TestNormalizeOptionalString(t *testing.T) {
	t.Parallel()

	t.Run("nil stays nil", func(t *testing.T) {
		t.Parallel()

		if got := normalizeOptionalString(nil); got != nil {
			t.Fatalf("normalizeOptionalString(nil) = %q, want nil", *got)
		}
	})

	t.Run("blank trims to nil", func(t *testing.T) {
		t.Parallel()

		value := "   "
		if got := normalizeOptionalString(&value); got != nil {
			t.Fatalf("normalizeOptionalString(blank) = %q, want nil", *got)
		}
	})

	t.Run("non blank returns trimmed value", func(t *testing.T) {
		t.Parallel()

		value := "  session-42  "
		got := normalizeOptionalString(&value)
		if got == nil {
			t.Fatal("normalizeOptionalString(non-blank) = nil, want value")
		}
		if *got != "session-42" {
			t.Fatalf("normalizeOptionalString(non-blank) = %q, want %q", *got, "session-42")
		}
	})
}
