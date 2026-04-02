package ticket

import (
	"strconv"
	"strings"
	"time"

	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/google/uuid"
)

const (
	defaultCreatedBy        = "user:api"
	defaultIdentifierPrefix = "ASE"
)

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func parseIdentifierSequence(identifier string) (int, bool) {
	if !strings.HasPrefix(identifier, defaultIdentifierPrefix+"-") {
		return 0, false
	}

	value, err := strconv.Atoi(strings.TrimPrefix(identifier, defaultIdentifierPrefix+"-"))
	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

func resolveCreatedBy(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return defaultCreatedBy
	}

	return strings.TrimSpace(raw)
}

func optionalUUIDPointerEqual(left *uuid.UUID, right *uuid.UUID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func selectTicketHookDefinitions(hooks infrahook.TicketHooks, hookName infrahook.TicketHookName) []infrahook.Definition {
	switch hookName {
	case infrahook.TicketHookOnClaim:
		return hooks.OnClaim
	case infrahook.TicketHookOnStart:
		return hooks.OnStart
	case infrahook.TicketHookOnComplete:
		return hooks.OnComplete
	case infrahook.TicketHookOnDone:
		return hooks.OnDone
	case infrahook.TicketHookOnError:
		return hooks.OnError
	case infrahook.TicketHookOnCancel:
		return hooks.OnCancel
	default:
		return nil
	}
}
