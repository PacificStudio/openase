package ticketing

import (
	"testing"

	"github.com/google/uuid"
)

func TestClassifyStatusChangeRunDisposition(t *testing.T) {
	pickupA := uuid.New()
	pickupB := uuid.New()
	finish := uuid.New()
	other := uuid.New()

	tests := []struct {
		name          string
		hasCurrentRun bool
		nextStatusID  uuid.UUID
		want          StatusChangeRunDisposition
	}{
		{
			name:          "retain when ticket has no current run",
			hasCurrentRun: false,
			nextStatusID:  other,
			want:          StatusChangeRunDispositionRetain,
		},
		{
			name:          "retain within pickup set",
			hasCurrentRun: true,
			nextStatusID:  pickupB,
			want:          StatusChangeRunDispositionRetain,
		},
		{
			name:          "release done for finish status",
			hasCurrentRun: true,
			nextStatusID:  finish,
			want:          StatusChangeRunDispositionDone,
		},
		{
			name:          "release cancel outside pickup and finish",
			hasCurrentRun: true,
			nextStatusID:  other,
			want:          StatusChangeRunDispositionCancel,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ClassifyStatusChangeRunDisposition(
				test.hasCurrentRun,
				test.nextStatusID,
				[]uuid.UUID{pickupA, pickupB},
				[]uuid.UUID{finish},
			)
			if got != test.want {
				t.Fatalf("ClassifyStatusChangeRunDisposition() = %q, want %q", got, test.want)
			}
		})
	}
}
