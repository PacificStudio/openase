package humanauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entauthauditevent "github.com/BetterAndBetterII/openase/ent/authauditevent"
	entbrowsersession "github.com/BetterAndBetterII/openase/ent/browsersession"
	entchatconversation "github.com/BetterAndBetterII/openase/ent/chatconversation"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entnotificationchannel "github.com/BetterAndBetterII/openase/ent/notificationchannel"
	entnotificationrule "github.com/BetterAndBetterII/openase/ent/notificationrule"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entrolebinding "github.com/BetterAndBetterII/openase/ent/rolebinding"
	entscheduledjob "github.com/BetterAndBetterII/openase/ent/scheduledjob"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entuser "github.com/BetterAndBetterII/openase/ent/user"
	entusergroupmembership "github.com/BetterAndBetterII/openase/ent/usergroupmembership"
	entuseridentity "github.com/BetterAndBetterII/openase/ent/useridentity"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
)

type Repository struct {
	client *ent.Client
}

var (
	ErrRoleBindingNotFound      = errors.New("role binding not found")
	ErrRoleBindingUserNotFound  = errors.New("role binding user not found")
	ErrRoleBindingUserAmbiguous = errors.New("role binding user lookup is ambiguous")
	ErrUserNotFound             = errors.New("user not found")
	ErrOIDCIdentityConflict     = errors.New("oidc identity conflicts with existing cached user")
)

type ListRoleBindingsFilter struct {
	ScopeKind *domain.ScopeKind
	ScopeID   *string
}

type CreateBrowserSessionInput struct {
	UserID        uuid.UUID
	SessionHash   string
	DeviceKind    domain.SessionDeviceKind
	DeviceOS      string
	DeviceBrowser string
	DeviceLabel   string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	CSRFSecret    string
	UserAgentHash string
	IPPrefix      string
}

type CreateAuthAuditEventInput struct {
	UserID    *uuid.UUID
	SessionID *uuid.UUID
	ActorID   string
	EventType domain.AuthAuditEventType
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

func NewEntRepository(client *ent.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) UpsertUserFromOIDC(
	ctx context.Context,
	profile domain.OIDCProfile,
) (domain.User, domain.UserIdentity, []domain.UserGroupMembership, error) {
	user, identity, memberships, err := r.upsertUserFromOIDC(ctx, profile)
	if err == nil {
		return user, identity, memberships, nil
	}
	if ent.IsConstraintError(err) {
		return r.upsertUserFromOIDC(ctx, profile)
	}
	return domain.User{}, domain.UserIdentity{}, nil, err
}

func (r *Repository) upsertUserFromOIDC(
	ctx context.Context,
	profile domain.OIDCProfile,
) (domain.User, domain.UserIdentity, []domain.UserGroupMembership, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("start user sync transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	identityItem, err := tx.UserIdentity.Query().
		Where(
			entuseridentity.IssuerEQ(strings.TrimSpace(profile.Issuer)),
			entuseridentity.SubjectEQ(strings.TrimSpace(profile.Subject)),
		).
		Only(ctx)
	userDisplayName := strings.TrimSpace(profile.DisplayName)
	if userDisplayName == "" {
		userDisplayName = strings.TrimSpace(profile.Username)
	}
	if userDisplayName == "" {
		userDisplayName = strings.TrimSpace(profile.Email)
	}
	normalizedEmail := strings.ToLower(strings.TrimSpace(profile.Email))
	now := time.Now().UTC()
	var userItem *ent.User
	switch {
	case ent.IsNotFound(err):
		if err := ensureNoUnsupportedIdentityMerge(ctx, tx, normalizedEmail, strings.TrimSpace(profile.Issuer), strings.TrimSpace(profile.Subject)); err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, err
		}
		userItem, err = tx.User.Create().
			SetStatus(entuser.StatusActive).
			SetPrimaryEmail(normalizedEmail).
			SetDisplayName(userDisplayName).
			SetAvatarURL(strings.TrimSpace(profile.AvatarURL)).
			SetLastLoginAt(now).
			Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("create user: %w", err)
		}
		identityItem, err = tx.UserIdentity.Create().
			SetUserID(userItem.ID).
			SetIssuer(strings.TrimSpace(profile.Issuer)).
			SetSubject(strings.TrimSpace(profile.Subject)).
			SetEmail(normalizedEmail).
			SetEmailVerified(profile.EmailVerified).
			SetClaimsVersion(1).
			SetRawClaimsJSON(strings.TrimSpace(profile.RawClaimsJSON)).
			SetLastSyncedAt(now).
			Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("create user identity: %w", err)
		}
	case err != nil:
		return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("query user identity: %w", err)
	default:
		userItem, err = tx.User.Get(ctx, identityItem.UserID)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("load identity user: %w", err)
		}
		identityChanged := userItem.PrimaryEmail != normalizedEmail ||
			userItem.DisplayName != userDisplayName ||
			userItem.AvatarURL != strings.TrimSpace(profile.AvatarURL) ||
			identityItem.Email != normalizedEmail ||
			identityItem.EmailVerified != profile.EmailVerified ||
			identityItem.RawClaimsJSON != strings.TrimSpace(profile.RawClaimsJSON)
		if identityChanged || !timestampsEqual(userItem.LastLoginAt, &now) {
			userItem, err = tx.User.UpdateOneID(userItem.ID).
				SetPrimaryEmail(normalizedEmail).
				SetDisplayName(userDisplayName).
				SetAvatarURL(strings.TrimSpace(profile.AvatarURL)).
				SetLastLoginAt(now).
				Save(ctx)
			if err != nil {
				return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("update user: %w", err)
			}
		}
		identityBuilder := tx.UserIdentity.UpdateOneID(identityItem.ID).
			SetEmail(normalizedEmail).
			SetEmailVerified(profile.EmailVerified).
			SetRawClaimsJSON(strings.TrimSpace(profile.RawClaimsJSON)).
			SetLastSyncedAt(now)
		if identityChanged {
			identityBuilder = identityBuilder.AddClaimsVersion(1)
		}
		identityItem, err = identityBuilder.Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("update user identity: %w", err)
		}
	}

	memberships, err := syncUserGroupMemberships(ctx, tx, userItem.ID, strings.TrimSpace(profile.Issuer), profile.Groups, now)
	if err != nil {
		return domain.User{}, domain.UserIdentity{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("commit user sync: %w", err)
	}
	return mapUser(userItem), mapUserIdentity(identityItem), memberships, nil
}

func (r *Repository) CreateBrowserSession(ctx context.Context, input CreateBrowserSessionInput) (domain.BrowserSession, error) {
	deviceKind := strings.TrimSpace(string(input.DeviceKind))
	if deviceKind == "" {
		deviceKind = string(domain.SessionDeviceKindUnknown)
	}
	item, err := r.client.BrowserSession.Create().
		SetUserID(input.UserID).
		SetSessionHash(strings.TrimSpace(input.SessionHash)).
		SetDeviceKind(deviceKind).
		SetDeviceOs(strings.TrimSpace(input.DeviceOS)).
		SetDeviceBrowser(strings.TrimSpace(input.DeviceBrowser)).
		SetDeviceLabel(strings.TrimSpace(input.DeviceLabel)).
		SetExpiresAt(input.ExpiresAt.UTC()).
		SetIdleExpiresAt(input.IdleExpiresAt.UTC()).
		SetCsrfSecret(strings.TrimSpace(input.CSRFSecret)).
		SetUserAgentHash(strings.TrimSpace(input.UserAgentHash)).
		SetIPPrefix(strings.TrimSpace(input.IPPrefix)).
		Save(ctx)
	if err != nil {
		return domain.BrowserSession{}, fmt.Errorf("create browser session: %w", err)
	}
	return mapBrowserSession(item), nil
}

func (r *Repository) GetBrowserSession(ctx context.Context, id uuid.UUID) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.Get(ctx, id)
	if err != nil {
		return domain.BrowserSession{}, fmt.Errorf("get browser session: %w", err)
	}
	return mapBrowserSession(item), nil
}

func (r *Repository) GetBrowserSessionByHash(ctx context.Context, sessionHash string) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.Query().
		Where(entbrowsersession.SessionHashEQ(strings.TrimSpace(sessionHash))).
		Only(ctx)
	if err != nil {
		return domain.BrowserSession{}, fmt.Errorf("get browser session: %w", err)
	}
	return mapBrowserSession(item), nil
}

func (r *Repository) ListBrowserSessionsByUser(ctx context.Context, userID uuid.UUID) ([]domain.BrowserSession, error) {
	items, err := r.client.BrowserSession.Query().
		Where(entbrowsersession.UserIDEQ(userID)).
		Order(
			ent.Desc(entbrowsersession.FieldUpdatedAt),
			ent.Desc(entbrowsersession.FieldCreatedAt),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list browser sessions: %w", err)
	}
	result := make([]domain.BrowserSession, 0, len(items))
	for _, item := range items {
		result = append(result, mapBrowserSession(item))
	}
	return result, nil
}

func (r *Repository) TouchBrowserSession(
	ctx context.Context,
	id uuid.UUID,
	expiresAt time.Time,
	idleExpiresAt time.Time,
) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.UpdateOneID(id).
		SetExpiresAt(expiresAt.UTC()).
		SetIdleExpiresAt(idleExpiresAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.BrowserSession{}, fmt.Errorf("touch browser session: %w", err)
	}
	return mapBrowserSession(item), nil
}

func (r *Repository) RevokeBrowserSession(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	if err := r.client.BrowserSession.UpdateOneID(id).
		SetRevokedAt(revokedAt.UTC()).
		Exec(ctx); err != nil {
		return fmt.Errorf("revoke browser session: %w", err)
	}
	return nil
}

func (r *Repository) RevokeBrowserSessionsByUser(
	ctx context.Context,
	userID uuid.UUID,
	excludeSessionID *uuid.UUID,
	revokedAt time.Time,
) ([]domain.BrowserSession, error) {
	query := r.client.BrowserSession.Query().
		Where(
			entbrowsersession.UserIDEQ(userID),
			entbrowsersession.RevokedAtIsNil(),
		)
	if excludeSessionID != nil {
		query = query.Where(entbrowsersession.IDNEQ(*excludeSessionID))
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list browser sessions to revoke: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(items))
	result := make([]domain.BrowserSession, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
		mapped := mapBrowserSession(item)
		revokedAtUTC := revokedAt.UTC()
		mapped.RevokedAt = &revokedAtUTC
		result = append(result, mapped)
	}

	if _, err := r.client.BrowserSession.Update().
		Where(entbrowsersession.IDIn(ids...)).
		SetRevokedAt(revokedAt.UTC()).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("revoke browser sessions: %w", err)
	}
	return result, nil
}

func (r *Repository) CreateAuthAuditEvent(ctx context.Context, input CreateAuthAuditEventInput) (domain.AuthAuditEvent, error) {
	builder := r.client.AuthAuditEvent.Create().
		SetActorID(strings.TrimSpace(input.ActorID)).
		SetEventType(strings.TrimSpace(string(input.EventType))).
		SetMessage(strings.TrimSpace(input.Message)).
		SetMetadata(input.Metadata)
	if !input.CreatedAt.IsZero() {
		builder.SetCreatedAt(input.CreatedAt.UTC())
	}
	if input.UserID != nil {
		builder.SetUserID(*input.UserID)
	}
	if input.SessionID != nil {
		builder.SetSessionID(*input.SessionID)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.AuthAuditEvent{}, fmt.Errorf("create auth audit event: %w", err)
	}
	return mapAuthAuditEvent(item), nil
}

func (r *Repository) ListAuthAuditEventsByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]domain.AuthAuditEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := r.client.AuthAuditEvent.Query().
		Where(entauthauditevent.UserIDEQ(userID)).
		Order(ent.Desc(entauthauditevent.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list auth audit events: %w", err)
	}
	result := make([]domain.AuthAuditEvent, 0, len(items))
	for _, item := range items {
		result = append(result, mapAuthAuditEvent(item))
	}
	return result, nil
}

func (r *Repository) GetUser(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	item, err := r.client.User.Get(ctx, userID)
	switch {
	case ent.IsNotFound(err):
		return domain.User{}, ErrUserNotFound
	case err != nil:
		return domain.User{}, fmt.Errorf("get user: %w", err)
	default:
		return mapUser(item), nil
	}
}

func (r *Repository) GetPrimaryIdentity(ctx context.Context, userID uuid.UUID) (domain.UserIdentity, error) {
	item, err := r.client.UserIdentity.Query().
		Where(entuseridentity.UserIDEQ(userID)).
		Order(ent.Asc(entuseridentity.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return domain.UserIdentity{}, fmt.Errorf("get primary identity: %w", err)
	}
	return mapUserIdentity(item), nil
}

func (r *Repository) ListUserIdentities(ctx context.Context, userID uuid.UUID) ([]domain.UserIdentity, error) {
	items, err := r.client.UserIdentity.Query().
		Where(entuseridentity.UserIDEQ(userID)).
		Order(ent.Asc(entuseridentity.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user identities: %w", err)
	}
	result := make([]domain.UserIdentity, 0, len(items))
	for _, item := range items {
		result = append(result, mapUserIdentity(item))
	}
	return result, nil
}

func (r *Repository) ListUsers(ctx context.Context, filter domain.UserDirectoryFilter) ([]domain.User, error) {
	query := r.client.User.Query()
	switch filter.Status {
	case domain.UserDirectoryStatusActive:
		query = query.Where(entuser.StatusEQ(entuser.StatusActive))
	case domain.UserDirectoryStatusDisabled:
		query = query.Where(entuser.StatusEQ(entuser.StatusDisabled))
	}

	if trimmed := strings.TrimSpace(filter.Query); trimmed != "" {
		userIDs, err := r.searchUserIDs(ctx, trimmed)
		if err != nil {
			return nil, err
		}
		if len(userIDs) == 0 {
			return []domain.User{}, nil
		}
		query = query.Where(entuser.IDIn(userIDs...))
	}

	items, err := query.
		Order(ent.Desc(entuser.FieldLastLoginAt), ent.Desc(entuser.FieldUpdatedAt)).
		Limit(filter.Limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	result := make([]domain.User, 0, len(items))
	for _, item := range items {
		result = append(result, mapUser(item))
	}
	return result, nil
}

func (r *Repository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) (domain.User, error) {
	item, err := r.client.User.UpdateOneID(userID).
		SetStatus(entuser.Status(status)).
		Save(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.User{}, ErrUserNotFound
	case err != nil:
		return domain.User{}, fmt.Errorf("update user status: %w", err)
	default:
		return mapUser(item), nil
	}
}

func (r *Repository) ListUserGroups(ctx context.Context, userID uuid.UUID) ([]domain.UserGroupMembership, error) {
	items, err := r.client.UserGroupMembership.Query().
		Where(entusergroupmembership.UserIDEQ(userID)).
		Order(ent.Asc(entusergroupmembership.FieldGroupKey)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user groups: %w", err)
	}
	result := make([]domain.UserGroupMembership, 0, len(items))
	for _, item := range items {
		result = append(result, mapGroupMembership(item))
	}
	return result, nil
}

func (r *Repository) ListSubjectRoleBindings(
	ctx context.Context,
	userKeys []string,
	groupKeys []string,
) ([]domain.RoleBinding, error) {
	predicates := make([]predicate.RoleBinding, 0, 2)
	if len(userKeys) > 0 {
		predicates = append(predicates, entrolebinding.And(
			entrolebinding.SubjectKindEQ(entrolebinding.SubjectKindUser),
			entrolebinding.SubjectKeyIn(userKeys...),
		))
	}
	if len(groupKeys) > 0 {
		predicates = append(predicates, entrolebinding.And(
			entrolebinding.SubjectKindEQ(entrolebinding.SubjectKindGroup),
			entrolebinding.SubjectKeyIn(groupKeys...),
		))
	}
	if len(predicates) == 0 {
		return []domain.RoleBinding{}, nil
	}
	items, err := r.client.RoleBinding.Query().
		Where(entrolebinding.Or(predicates...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subject role bindings: %w", err)
	}
	result := make([]domain.RoleBinding, 0, len(items))
	for _, item := range items {
		result = append(result, mapRoleBinding(item))
	}
	return result, nil
}

func (r *Repository) ListRoleBindings(ctx context.Context, filter ListRoleBindingsFilter) ([]domain.RoleBinding, error) {
	query := r.client.RoleBinding.Query()
	if filter.ScopeKind != nil {
		query = query.Where(entrolebinding.ScopeKindEQ(entrolebinding.ScopeKind(*filter.ScopeKind)))
	}
	if filter.ScopeID != nil {
		query = query.Where(entrolebinding.ScopeIDEQ(strings.TrimSpace(*filter.ScopeID)))
	}
	items, err := query.Order(ent.Asc(entrolebinding.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list role bindings: %w", err)
	}
	result := make([]domain.RoleBinding, 0, len(items))
	for _, item := range items {
		result = append(result, mapRoleBinding(item))
	}
	return result, nil
}

func (r *Repository) CreateRoleBinding(ctx context.Context, input domain.RoleBinding) (domain.RoleBinding, error) {
	return r.createRoleBinding(ctx, input)
}

func (r *Repository) createRoleBinding(ctx context.Context, input domain.RoleBinding) (domain.RoleBinding, error) {
	builder := r.client.RoleBinding.Create().
		SetScopeKind(entrolebinding.ScopeKind(input.ScopeKind)).
		SetScopeID(strings.TrimSpace(input.ScopeID)).
		SetSubjectKind(entrolebinding.SubjectKind(input.SubjectKind)).
		SetSubjectKey(strings.ToLower(strings.TrimSpace(input.SubjectKey))).
		SetRoleKey(string(input.RoleKey)).
		SetGrantedBy(strings.TrimSpace(input.GrantedBy))
	if input.ExpiresAt != nil {
		builder = builder.SetExpiresAt(input.ExpiresAt.UTC())
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.RoleBinding{}, fmt.Errorf("create role binding: %w", err)
	}
	return mapRoleBinding(item), nil
}

func (r *Repository) DeleteRoleBinding(ctx context.Context, id uuid.UUID) error {
	if err := r.client.RoleBinding.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("delete role binding: %w", err)
	}
	return nil
}

func (r *Repository) ResolveRoleBindingUser(ctx context.Context, raw string) (domain.User, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return domain.User{}, ErrRoleBindingUserNotFound
	}
	if parsed, err := uuid.Parse(normalized); err == nil {
		item, getErr := r.client.User.Query().Where(entuser.IDEQ(parsed)).Only(ctx)
		switch {
		case ent.IsNotFound(getErr):
			return domain.User{}, ErrRoleBindingUserNotFound
		case getErr != nil:
			return domain.User{}, fmt.Errorf("resolve role binding user by id: %w", getErr)
		default:
			return mapUser(item), nil
		}
	}

	primaryMatches, err := r.client.User.Query().
		Where(entuser.PrimaryEmailEQ(normalized)).
		All(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("resolve role binding user by primary email: %w", err)
	}
	if len(primaryMatches) == 1 {
		return mapUser(primaryMatches[0]), nil
	}
	if len(primaryMatches) > 1 {
		return domain.User{}, ErrRoleBindingUserAmbiguous
	}

	identityMatches, err := r.client.UserIdentity.Query().
		Where(entuseridentity.EmailEQ(normalized)).
		All(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("resolve role binding user by identity email: %w", err)
	}
	if len(identityMatches) == 0 {
		return domain.User{}, ErrRoleBindingUserNotFound
	}
	userIDs := make(map[uuid.UUID]struct{}, len(identityMatches))
	for _, match := range identityMatches {
		userIDs[match.UserID] = struct{}{}
	}
	if len(userIDs) != 1 {
		return domain.User{}, ErrRoleBindingUserAmbiguous
	}
	for userID := range userIDs {
		item, getErr := r.client.User.Query().Where(entuser.IDEQ(userID)).Only(ctx)
		switch {
		case ent.IsNotFound(getErr):
			return domain.User{}, ErrRoleBindingUserNotFound
		case getErr != nil:
			return domain.User{}, fmt.Errorf("load role binding user from identity: %w", getErr)
		default:
			return mapUser(item), nil
		}
	}
	return domain.User{}, ErrRoleBindingUserNotFound
}

func (r *Repository) ListInstanceRoleBindings(ctx context.Context) ([]domain.InstanceRoleBinding, error) {
	items, err := r.ListRoleBindings(ctx, ListRoleBindingsFilter{
		ScopeKind: scopeKindPointer(domain.ScopeKindInstance),
		ScopeID:   stringPointer(""),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.InstanceRoleBinding, 0, len(items))
	for _, item := range items {
		parsed, parseErr := domain.ParseInstanceRoleBinding(item)
		if parseErr != nil {
			return nil, fmt.Errorf("list instance role bindings: %w", parseErr)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func (r *Repository) ListOrganizationRoleBindings(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]domain.OrganizationRoleBinding, error) {
	items, err := r.ListRoleBindings(ctx, ListRoleBindingsFilter{
		ScopeKind: scopeKindPointer(domain.ScopeKindOrganization),
		ScopeID:   stringPointer(organizationID.String()),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.OrganizationRoleBinding, 0, len(items))
	for _, item := range items {
		parsed, parseErr := domain.ParseOrganizationRoleBinding(item)
		if parseErr != nil {
			return nil, fmt.Errorf("list organization role bindings: %w", parseErr)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func (r *Repository) ListProjectRoleBindings(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRoleBinding, error) {
	items, err := r.ListRoleBindings(ctx, ListRoleBindingsFilter{
		ScopeKind: scopeKindPointer(domain.ScopeKindProject),
		ScopeID:   stringPointer(projectID.String()),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.ProjectRoleBinding, 0, len(items))
	for _, item := range items {
		parsed, parseErr := domain.ParseProjectRoleBinding(item)
		if parseErr != nil {
			return nil, fmt.Errorf("list project role bindings: %w", parseErr)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func (r *Repository) CreateInstanceRoleBinding(
	ctx context.Context,
	input domain.InstanceRoleBinding,
) (domain.InstanceRoleBinding, error) {
	item, err := r.createRoleBinding(ctx, input.Generic())
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	return domain.ParseInstanceRoleBinding(item)
}

func (r *Repository) CreateOrganizationRoleBinding(
	ctx context.Context,
	input domain.OrganizationRoleBinding,
) (domain.OrganizationRoleBinding, error) {
	item, err := r.createRoleBinding(ctx, input.Generic())
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	return domain.ParseOrganizationRoleBinding(item)
}

func (r *Repository) CreateProjectRoleBinding(
	ctx context.Context,
	input domain.ProjectRoleBinding,
) (domain.ProjectRoleBinding, error) {
	item, err := r.createRoleBinding(ctx, input.Generic())
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	return domain.ParseProjectRoleBinding(item)
}

func (r *Repository) UpdateInstanceRoleBinding(
	ctx context.Context,
	id uuid.UUID,
	input domain.UpdateInstanceRoleBinding,
) (domain.InstanceRoleBinding, error) {
	item, err := r.updateRoleBinding(
		ctx,
		id,
		domain.ScopeKindInstance,
		"",
		string(input.Subject.Kind),
		input.Subject.Key,
		string(input.RoleKey),
		input.GrantedBy,
		input.ExpiresAt,
	)
	if err != nil {
		return domain.InstanceRoleBinding{}, err
	}
	return domain.ParseInstanceRoleBinding(item)
}

func (r *Repository) UpdateOrganizationRoleBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	input domain.UpdateOrganizationRoleBinding,
) (domain.OrganizationRoleBinding, error) {
	item, err := r.updateRoleBinding(
		ctx,
		id,
		domain.ScopeKindOrganization,
		organizationID.String(),
		string(input.Subject.Kind),
		input.Subject.Key,
		string(input.RoleKey),
		input.GrantedBy,
		input.ExpiresAt,
	)
	if err != nil {
		return domain.OrganizationRoleBinding{}, err
	}
	return domain.ParseOrganizationRoleBinding(item)
}

func (r *Repository) UpdateProjectRoleBinding(
	ctx context.Context,
	projectID uuid.UUID,
	id uuid.UUID,
	input domain.UpdateProjectRoleBinding,
) (domain.ProjectRoleBinding, error) {
	item, err := r.updateRoleBinding(
		ctx,
		id,
		domain.ScopeKindProject,
		projectID.String(),
		string(input.Subject.Kind),
		input.Subject.Key,
		string(input.RoleKey),
		input.GrantedBy,
		input.ExpiresAt,
	)
	if err != nil {
		return domain.ProjectRoleBinding{}, err
	}
	return domain.ParseProjectRoleBinding(item)
}

func (r *Repository) DeleteInstanceRoleBinding(ctx context.Context, id uuid.UUID) error {
	return r.deleteRoleBindingInScope(ctx, id, domain.ScopeKindInstance, "")
}

func (r *Repository) DeleteOrganizationRoleBinding(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) error {
	return r.deleteRoleBindingInScope(ctx, id, domain.ScopeKindOrganization, organizationID.String())
}

func (r *Repository) DeleteProjectRoleBinding(ctx context.Context, projectID uuid.UUID, id uuid.UUID) error {
	return r.deleteRoleBindingInScope(ctx, id, domain.ScopeKindProject, projectID.String())
}

func (r *Repository) EnsureBootstrapRoleBinding(
	ctx context.Context,
	user domain.User,
	grantedBy string,
) (domain.RoleBinding, error) {
	item, err := r.client.RoleBinding.Query().
		Where(
			entrolebinding.ScopeKindEQ(entrolebinding.ScopeKindInstance),
			entrolebinding.ScopeIDEQ(""),
			entrolebinding.SubjectKindEQ(entrolebinding.SubjectKindUser),
			entrolebinding.SubjectKeyEQ(user.ID.String()),
			entrolebinding.RoleKeyEQ(string(domain.RoleInstanceAdmin)),
		).
		Only(ctx)
	if err == nil {
		return mapRoleBinding(item), nil
	}
	if !ent.IsNotFound(err) {
		return domain.RoleBinding{}, fmt.Errorf("query bootstrap role binding: %w", err)
	}
	return r.CreateRoleBinding(ctx, domain.RoleBinding{
		ScopeKind:   domain.ScopeKindInstance,
		ScopeID:     "",
		SubjectKind: domain.SubjectKindUser,
		SubjectKey:  user.ID.String(),
		RoleKey:     domain.RoleInstanceAdmin,
		GrantedBy:   grantedBy,
	})
}

func (r *Repository) ResolveProjectOrganization(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Project.Query().
		Where(entproject.IDEQ(projectID)).
		Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve project organization: %w", err)
	}
	return item.OrganizationID, nil
}

func (r *Repository) ResolveProjectFromTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Ticket.Query().Where(entticket.IDEQ(ticketID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve ticket project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromWorkflow(ctx context.Context, workflowID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Workflow.Query().Where(entworkflow.IDEQ(workflowID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve workflow project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromSkill(ctx context.Context, skillID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Skill.Query().Where(entskill.IDEQ(skillID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve skill project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromStatus(ctx context.Context, statusID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.TicketStatus.Query().Where(entticketstatus.IDEQ(statusID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve status project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromAgent(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Agent.Query().Where(entagent.IDEQ(agentID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve agent project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromScheduledJob(ctx context.Context, jobID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.ScheduledJob.Query().Where(entscheduledjob.IDEQ(jobID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve scheduled job project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromNotificationRule(ctx context.Context, ruleID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.NotificationRule.Query().Where(entnotificationrule.IDEQ(ruleID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve notification rule project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveProjectFromConversation(ctx context.Context, conversationID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.ChatConversation.Query().Where(entchatconversation.IDEQ(conversationID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve conversation project: %w", err)
	}
	return item.ProjectID, nil
}

func (r *Repository) ResolveOrganizationFromMachine(ctx context.Context, machineID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.Machine.Query().Where(entmachine.IDEQ(machineID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve machine organization: %w", err)
	}
	return item.OrganizationID, nil
}

func (r *Repository) ResolveOrganizationFromProvider(ctx context.Context, providerID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.AgentProvider.Query().Where(entagentprovider.IDEQ(providerID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve provider organization: %w", err)
	}
	return item.OrganizationID, nil
}

func (r *Repository) ResolveOrganizationFromChannel(ctx context.Context, channelID uuid.UUID) (uuid.UUID, error) {
	item, err := r.client.NotificationChannel.Query().Where(entnotificationchannel.IDEQ(channelID)).Only(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve channel organization: %w", err)
	}
	return item.OrganizationID, nil
}

func (r *Repository) CountApprovalPolicies(ctx context.Context) (int, error) {
	count, err := r.client.ApprovalPolicyRule.Query().Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count approval policies: %w", err)
	}
	return count, nil
}

func (r *Repository) searchUserIDs(ctx context.Context, query string) ([]uuid.UUID, error) {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return nil, nil
	}

	userIDs := map[uuid.UUID]struct{}{}
	if parsedID, err := uuid.Parse(normalized); err == nil {
		userIDs[parsedID] = struct{}{}
	}

	userMatches, err := r.client.User.Query().
		Where(
			entuser.Or(
				entuser.PrimaryEmailContainsFold(normalized),
				entuser.DisplayNameContainsFold(normalized),
			),
		).
		IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("search users by primary fields: %w", err)
	}
	for _, id := range userMatches {
		userIDs[id] = struct{}{}
	}

	identityMatches, err := r.client.UserIdentity.Query().
		Where(
			entuseridentity.Or(
				entuseridentity.EmailContainsFold(normalized),
				entuseridentity.IssuerContainsFold(normalized),
				entuseridentity.SubjectContainsFold(normalized),
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("search users by identity fields: %w", err)
	}
	for _, item := range identityMatches {
		userIDs[item.UserID] = struct{}{}
	}

	result := make([]uuid.UUID, 0, len(userIDs))
	for id := range userIDs {
		result = append(result, id)
	}
	return result, nil
}

func ensureNoUnsupportedIdentityMerge(
	ctx context.Context,
	tx *ent.Tx,
	normalizedEmail string,
	issuer string,
	subject string,
) error {
	if normalizedEmail == "" {
		return nil
	}

	existingByPrimaryEmail, err := tx.User.Query().
		Where(entuser.PrimaryEmailEQ(normalizedEmail)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("check existing user email: %w", err)
	}
	if len(existingByPrimaryEmail) > 0 {
		return fmt.Errorf("%w: matching primary email %q already belongs to another user; automatic link/unlink/merge is not supported", ErrOIDCIdentityConflict, normalizedEmail)
	}

	existingIdentity, err := tx.UserIdentity.Query().
		Where(
			entuseridentity.EmailEQ(normalizedEmail),
			entuseridentity.Or(
				entuseridentity.IssuerNEQ(strings.TrimSpace(issuer)),
				entuseridentity.SubjectNEQ(strings.TrimSpace(subject)),
			),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check existing identity email: %w", err)
	}
	if existingIdentity {
		return fmt.Errorf("%w: matching identity email %q already belongs to another subject; automatic link/unlink/merge is not supported", ErrOIDCIdentityConflict, normalizedEmail)
	}
	return nil
}

func syncUserGroupMemberships(
	ctx context.Context,
	tx *ent.Tx,
	userID uuid.UUID,
	issuer string,
	groups []domain.Group,
	now time.Time,
) ([]domain.UserGroupMembership, error) {
	if _, err := tx.UserGroupMembership.Delete().
		Where(
			entusergroupmembership.UserIDEQ(userID),
			entusergroupmembership.IssuerEQ(strings.TrimSpace(issuer)),
		).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("replace group memberships: %w", err)
	}

	memberships := make([]domain.UserGroupMembership, 0, len(groups))
	for _, group := range groups {
		if strings.TrimSpace(group.Key) == "" {
			continue
		}
		item, err := tx.UserGroupMembership.Create().
			SetUserID(userID).
			SetIssuer(strings.TrimSpace(issuer)).
			SetGroupKey(strings.ToLower(strings.TrimSpace(group.Key))).
			SetGroupName(strings.TrimSpace(group.Name)).
			SetLastSyncedAt(now).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create group membership: %w", err)
		}
		memberships = append(memberships, mapGroupMembership(item))
	}
	return memberships, nil
}

func timestampsEqual(left *time.Time, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.UTC().Equal(right.UTC())
	}
}

func mapUser(item *ent.User) domain.User {
	return domain.User{
		ID:           item.ID,
		Status:       domain.UserStatus(item.Status),
		PrimaryEmail: item.PrimaryEmail,
		DisplayName:  item.DisplayName,
		AvatarURL:    item.AvatarURL,
		LastLoginAt:  item.LastLoginAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func mapUserIdentity(item *ent.UserIdentity) domain.UserIdentity {
	return domain.UserIdentity{
		ID:            item.ID,
		UserID:        item.UserID,
		Issuer:        item.Issuer,
		Subject:       item.Subject,
		Email:         item.Email,
		EmailVerified: item.EmailVerified,
		ClaimsVersion: item.ClaimsVersion,
		RawClaimsJSON: item.RawClaimsJSON,
		LastSyncedAt:  item.LastSyncedAt,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func mapGroupMembership(item *ent.UserGroupMembership) domain.UserGroupMembership {
	return domain.UserGroupMembership{
		ID:           item.ID,
		UserID:       item.UserID,
		Issuer:       item.Issuer,
		GroupKey:     item.GroupKey,
		GroupName:    item.GroupName,
		LastSyncedAt: item.LastSyncedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func mapBrowserSession(item *ent.BrowserSession) domain.BrowserSession {
	return domain.BrowserSession{
		ID:            item.ID,
		UserID:        item.UserID,
		SessionHash:   item.SessionHash,
		DeviceKind:    domain.SessionDeviceKind(item.DeviceKind),
		DeviceOS:      item.DeviceOs,
		DeviceBrowser: item.DeviceBrowser,
		DeviceLabel:   item.DeviceLabel,
		ExpiresAt:     item.ExpiresAt,
		IdleExpiresAt: item.IdleExpiresAt,
		CSRFSecret:    item.CsrfSecret,
		UserAgentHash: item.UserAgentHash,
		IPPrefix:      item.IPPrefix,
		RevokedAt:     item.RevokedAt,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func mapAuthAuditEvent(item *ent.AuthAuditEvent) domain.AuthAuditEvent {
	return domain.AuthAuditEvent{
		ID:        item.ID,
		UserID:    item.UserID,
		SessionID: item.SessionID,
		ActorID:   item.ActorID,
		EventType: domain.AuthAuditEventType(item.EventType),
		Message:   item.Message,
		Metadata:  item.Metadata,
		CreatedAt: item.CreatedAt,
	}
}

func mapRoleBinding(item *ent.RoleBinding) domain.RoleBinding {
	return domain.RoleBinding{
		ID:          item.ID,
		ScopeKind:   domain.ScopeKind(item.ScopeKind),
		ScopeID:     item.ScopeID,
		SubjectKind: domain.SubjectKind(item.SubjectKind),
		SubjectKey:  item.SubjectKey,
		RoleKey:     domain.RoleKey(item.RoleKey),
		GrantedBy:   item.GrantedBy,
		ExpiresAt:   item.ExpiresAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func (r *Repository) updateRoleBinding(
	ctx context.Context,
	id uuid.UUID,
	scopeKind domain.ScopeKind,
	scopeID string,
	subjectKind string,
	subjectKey string,
	roleKey string,
	grantedBy string,
	expiresAt *time.Time,
) (domain.RoleBinding, error) {
	builder := r.client.RoleBinding.UpdateOneID(id).
		Where(
			entrolebinding.ScopeKindEQ(entrolebinding.ScopeKind(scopeKind)),
			entrolebinding.ScopeIDEQ(strings.TrimSpace(scopeID)),
		).
		SetSubjectKind(entrolebinding.SubjectKind(subjectKind)).
		SetSubjectKey(strings.ToLower(strings.TrimSpace(subjectKey))).
		SetRoleKey(strings.TrimSpace(roleKey)).
		SetGrantedBy(strings.TrimSpace(grantedBy))
	if expiresAt != nil {
		builder = builder.SetExpiresAt(expiresAt.UTC())
	} else {
		builder = builder.ClearExpiresAt()
	}
	item, err := builder.Save(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.RoleBinding{}, ErrRoleBindingNotFound
	case err != nil:
		return domain.RoleBinding{}, fmt.Errorf("update role binding: %w", err)
	default:
		return mapRoleBinding(item), nil
	}
}

func (r *Repository) deleteRoleBindingInScope(
	ctx context.Context,
	id uuid.UUID,
	scopeKind domain.ScopeKind,
	scopeID string,
) error {
	affected, err := r.client.RoleBinding.Delete().
		Where(
			entrolebinding.IDEQ(id),
			entrolebinding.ScopeKindEQ(entrolebinding.ScopeKind(scopeKind)),
			entrolebinding.ScopeIDEQ(strings.TrimSpace(scopeID)),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete role binding: %w", err)
	}
	if affected == 0 {
		return ErrRoleBindingNotFound
	}
	return nil
}

func scopeKindPointer(value domain.ScopeKind) *domain.ScopeKind {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
