package humanauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
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

type ListRoleBindingsFilter struct {
	ScopeKind *domain.ScopeKind
	ScopeID   *string
}

type CreateBrowserSessionInput struct {
	UserID        uuid.UUID
	SessionHash   string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	CSRFSecret    string
	UserAgentHash string
	IPPrefix      string
}

func NewEntRepository(client *ent.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) UpsertUserFromOIDC(
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
		userItem, err = tx.User.UpdateOneID(userItem.ID).
			SetPrimaryEmail(normalizedEmail).
			SetDisplayName(userDisplayName).
			SetAvatarURL(strings.TrimSpace(profile.AvatarURL)).
			SetLastLoginAt(now).
			Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("update user: %w", err)
		}
		identityItem, err = tx.UserIdentity.UpdateOneID(identityItem.ID).
			SetEmail(normalizedEmail).
			SetEmailVerified(profile.EmailVerified).
			SetRawClaimsJSON(strings.TrimSpace(profile.RawClaimsJSON)).
			SetLastSyncedAt(now).
			AddClaimsVersion(1).
			Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("update user identity: %w", err)
		}
	}

	if _, err := tx.UserGroupMembership.Delete().
		Where(
			entusergroupmembership.UserIDEQ(userItem.ID),
			entusergroupmembership.IssuerEQ(strings.TrimSpace(profile.Issuer)),
		).
		Exec(ctx); err != nil {
		return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("replace group memberships: %w", err)
	}

	memberships := make([]domain.UserGroupMembership, 0, len(profile.Groups))
	for _, group := range profile.Groups {
		if strings.TrimSpace(group.Key) == "" {
			continue
		}
		item, err := tx.UserGroupMembership.Create().
			SetUserID(userItem.ID).
			SetIssuer(strings.TrimSpace(profile.Issuer)).
			SetGroupKey(strings.ToLower(strings.TrimSpace(group.Key))).
			SetGroupName(strings.TrimSpace(group.Name)).
			SetLastSyncedAt(now).
			Save(ctx)
		if err != nil {
			return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("create group membership: %w", err)
		}
		memberships = append(memberships, mapGroupMembership(item))
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, domain.UserIdentity{}, nil, fmt.Errorf("commit user sync: %w", err)
	}
	return mapUser(userItem), mapUserIdentity(identityItem), memberships, nil
}

func (r *Repository) CreateBrowserSession(ctx context.Context, input CreateBrowserSessionInput) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.Create().
		SetUserID(input.UserID).
		SetSessionHash(strings.TrimSpace(input.SessionHash)).
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

func (r *Repository) GetBrowserSessionByHash(ctx context.Context, sessionHash string) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.Query().
		Where(entbrowsersession.SessionHashEQ(strings.TrimSpace(sessionHash))).
		Only(ctx)
	if err != nil {
		return domain.BrowserSession{}, fmt.Errorf("get browser session: %w", err)
	}
	return mapBrowserSession(item), nil
}

func (r *Repository) TouchBrowserSession(ctx context.Context, id uuid.UUID, idleExpiresAt time.Time) (domain.BrowserSession, error) {
	item, err := r.client.BrowserSession.UpdateOneID(id).
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

func (r *Repository) GetUser(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	item, err := r.client.User.Get(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return mapUser(item), nil
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
