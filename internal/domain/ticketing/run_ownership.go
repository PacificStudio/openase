package ticketing

import (
	"slices"

	"github.com/google/uuid"
)

type StatusChangeRunDisposition string

const (
	StatusChangeRunDispositionRetain StatusChangeRunDisposition = "retain"
	StatusChangeRunDispositionDone   StatusChangeRunDisposition = "release_done"
	StatusChangeRunDispositionCancel StatusChangeRunDisposition = "release_cancel"
)

func ClassifyStatusChangeRunDisposition(
	hasCurrentRun bool,
	nextStatusID uuid.UUID,
	pickupStatusIDs []uuid.UUID,
	finishStatusIDs []uuid.UUID,
) StatusChangeRunDisposition {
	if !hasCurrentRun {
		return StatusChangeRunDispositionRetain
	}
	if slices.Contains(finishStatusIDs, nextStatusID) {
		return StatusChangeRunDispositionDone
	}
	if slices.Contains(pickupStatusIDs, nextStatusID) {
		return StatusChangeRunDispositionRetain
	}
	return StatusChangeRunDispositionCancel
}
