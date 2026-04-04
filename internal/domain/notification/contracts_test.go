package notification

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestChannelUsageConflictWrapsErrChannelInUse(t *testing.T) {
	conflict := &ChannelUsageConflict{ChannelID: uuid.New()}
	if !errors.Is(conflict, ErrChannelInUse) {
		t.Fatal("ChannelUsageConflict should wrap ErrChannelInUse")
	}
	if conflict.Error() != ErrChannelInUse.Error() {
		t.Fatalf("ChannelUsageConflict.Error() = %q", conflict.Error())
	}
}

func TestChannelUsageConflictNilReceiver(t *testing.T) {
	var conflict *ChannelUsageConflict
	if got := conflict.Error(); got != "" {
		t.Fatalf("nil ChannelUsageConflict.Error() = %q", got)
	}
	if conflict.Unwrap() != nil {
		t.Fatal("nil ChannelUsageConflict.Unwrap() should be nil")
	}
}
