package chat

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

const (
	// InstanceAdminProjectConversationUserID is the stable non-OIDC owner used
	// for persistent project conversations when browser login is disabled.
	InstanceAdminProjectConversationUserID UserID = "instance-admin"
	// LegacyLocalProjectConversationUserID is the previous auth-disabled owner
	// marker that now converges to instance-admin.
	LegacyLocalProjectConversationUserID UserID = "local-user:default"
	browserSessionProjectConversationUserIDPrefix = "browser-session:"
	humanProjectConversationUserIDPrefix          = "user:"
)

// LocalProjectConversationUserID is kept as a compatibility alias for older
// call sites; new code should prefer InstanceAdminProjectConversationUserID.
const LocalProjectConversationUserID = InstanceAdminProjectConversationUserID

type projectConversationOwnerKind string

const (
	projectConversationOwnerKindHumanStable projectConversationOwnerKind = "human_stable"
	projectConversationOwnerKindHumanLegacy projectConversationOwnerKind = "human_legacy"
	projectConversationOwnerKindInstance    projectConversationOwnerKind = "instance_admin"
	projectConversationOwnerKindLegacyLocal projectConversationOwnerKind = "legacy_local"
	projectConversationOwnerKindAnonymous   projectConversationOwnerKind = "anonymous"
	projectConversationOwnerKindBrowser     projectConversationOwnerKind = "browser_session"
	projectConversationOwnerKindUnknown     projectConversationOwnerKind = "unknown"
)

type parsedProjectConversationOwner struct {
	Raw         UserID
	Kind        projectConversationOwnerKind
	HumanUserID uuid.UUID
}

type ProjectConversationAccessHint struct {
	LegacyBrowserOwner               UserID
	AllowInstanceAdminLegacyAdoption bool
}

type projectConversationAccessHintContextKey struct{}

func WithProjectConversationAccessHint(
	ctx context.Context,
	hint ProjectConversationAccessHint,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := ProjectConversationAccessHint{
		AllowInstanceAdminLegacyAdoption: hint.AllowInstanceAdminLegacyAdoption,
	}
	if parsed, ok := parseProjectConversationOwnerLoose(hint.LegacyBrowserOwner.String()); ok &&
		parsed.Kind == projectConversationOwnerKindBrowser {
		normalized.LegacyBrowserOwner = parsed.Raw
	}
	if normalized.LegacyBrowserOwner == "" && !normalized.AllowInstanceAdminLegacyAdoption {
		return ctx
	}
	return context.WithValue(ctx, projectConversationAccessHintContextKey{}, normalized)
}

func projectConversationAccessHintFromContext(ctx context.Context) ProjectConversationAccessHint {
	if ctx == nil {
		return ProjectConversationAccessHint{}
	}
	hint, _ := ctx.Value(projectConversationAccessHintContextKey{}).(ProjectConversationAccessHint)
	return hint
}

func parseProjectConversationOwnerLoose(raw string) (parsedProjectConversationOwner, bool) {
	userID, err := ParseUserID(raw)
	if err != nil {
		return parsedProjectConversationOwner{}, false
	}
	trimmed := strings.TrimSpace(userID.String())
	switch {
	case trimmed == InstanceAdminProjectConversationUserID.String():
		return parsedProjectConversationOwner{Raw: InstanceAdminProjectConversationUserID, Kind: projectConversationOwnerKindInstance}, true
	case trimmed == LegacyLocalProjectConversationUserID.String():
		return parsedProjectConversationOwner{Raw: LegacyLocalProjectConversationUserID, Kind: projectConversationOwnerKindLegacyLocal}, true
	case trimmed == AnonymousUserID.String():
		return parsedProjectConversationOwner{Raw: AnonymousUserID, Kind: projectConversationOwnerKindAnonymous}, true
	case strings.HasPrefix(trimmed, humanProjectConversationUserIDPrefix):
		humanUserID, parseErr := uuid.Parse(strings.TrimPrefix(trimmed, humanProjectConversationUserIDPrefix))
		if parseErr != nil {
			return parsedProjectConversationOwner{Raw: userID, Kind: projectConversationOwnerKindUnknown}, true
		}
		return parsedProjectConversationOwner{Raw: userID, Kind: projectConversationOwnerKindHumanStable, HumanUserID: humanUserID}, true
	case strings.HasPrefix(trimmed, browserSessionProjectConversationUserIDPrefix):
		sessionID, parseErr := uuid.Parse(strings.TrimPrefix(trimmed, browserSessionProjectConversationUserIDPrefix))
		if parseErr != nil {
			return parsedProjectConversationOwner{Raw: userID, Kind: projectConversationOwnerKindUnknown}, true
		}
		return parsedProjectConversationOwner{Raw: UserID(browserSessionProjectConversationUserIDPrefix + sessionID.String()), Kind: projectConversationOwnerKindBrowser}, true
	default:
		humanUserID, parseErr := uuid.Parse(trimmed)
		if parseErr == nil {
			return parsedProjectConversationOwner{Raw: userID, Kind: projectConversationOwnerKindHumanLegacy, HumanUserID: humanUserID}, true
		}
		return parsedProjectConversationOwner{Raw: userID, Kind: projectConversationOwnerKindUnknown}, true
	}
}

type projectConversationOwnerAccess struct {
	allowed        bool
	normalizedUser UserID
	needsMigration bool
}

func normalizeProjectConversationOwnerForCreate(userID UserID) UserID {
	parsed, ok := parseProjectConversationOwnerLoose(userID.String())
	if !ok {
		return InstanceAdminProjectConversationUserID
	}

	switch parsed.Kind {
	case projectConversationOwnerKindHumanStable:
		return parsed.Raw
	case projectConversationOwnerKindHumanLegacy:
		return UserID(humanProjectConversationUserIDPrefix + parsed.HumanUserID.String())
	case projectConversationOwnerKindInstance,
		projectConversationOwnerKindLegacyLocal,
		projectConversationOwnerKindAnonymous,
		projectConversationOwnerKindBrowser,
		projectConversationOwnerKindUnknown:
		return InstanceAdminProjectConversationUserID
	default:
		return InstanceAdminProjectConversationUserID
	}
}

func resolveProjectConversationOwnerAccess(
	requestedUser UserID,
	storedUser string,
	hint ProjectConversationAccessHint,
) projectConversationOwnerAccess {
	requested, ok := parseProjectConversationOwnerLoose(requestedUser.String())
	if !ok {
		return projectConversationOwnerAccess{}
	}
	stored, ok := parseProjectConversationOwnerLoose(storedUser)
	if !ok {
		return projectConversationOwnerAccess{}
	}

	switch requested.Kind {
	case projectConversationOwnerKindInstance:
		return projectConversationOwnerAccess{
			allowed:        true,
			normalizedUser: InstanceAdminProjectConversationUserID,
			needsMigration: stored.Raw != InstanceAdminProjectConversationUserID,
		}
	case projectConversationOwnerKindHumanStable:
		switch {
		case stored.Raw == requested.Raw:
			return projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: requested.Raw,
			}
		case stored.Kind == projectConversationOwnerKindHumanLegacy &&
			stored.HumanUserID == requested.HumanUserID:
			return projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: requested.Raw,
				needsMigration: true,
			}
		case hint.LegacyBrowserOwner != "" && stored.Raw == hint.LegacyBrowserOwner:
			return projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: requested.Raw,
				needsMigration: true,
			}
		case hint.AllowInstanceAdminLegacyAdoption && projectConversationOwnerEligibleForAdminAdoption(stored):
			return projectConversationOwnerAccess{
				allowed:        true,
				normalizedUser: requested.Raw,
				needsMigration: stored.Raw != requested.Raw,
			}
		default:
			return projectConversationOwnerAccess{}
		}
	default:
		if stored.Raw != requested.Raw {
			return projectConversationOwnerAccess{}
		}
		return projectConversationOwnerAccess{
			allowed:        true,
			normalizedUser: requested.Raw,
		}
	}
}

func projectConversationOwnerEligibleForAdminAdoption(owner parsedProjectConversationOwner) bool {
	switch owner.Kind {
	case projectConversationOwnerKindInstance,
		projectConversationOwnerKindLegacyLocal,
		projectConversationOwnerKindAnonymous,
		projectConversationOwnerKindUnknown:
		return true
	default:
		return false
	}
}
