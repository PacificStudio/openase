package accesscontrol

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entinstanceauthconfig "github.com/BetterAndBetterII/openase/ent/instanceauthconfig"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
)

const singletonScopeKey = "instance"

type StoredConfigRecord struct {
	Status                string
	IssuerURL             string
	ClientID              string
	ClientSecretEncrypted *iam.EncryptedSecret
	RedirectMode          string
	FixedRedirectURL      string
	Scopes                []string
	EmailClaim            string
	NameClaim             string
	UsernameClaim         string
	GroupsClaim           string
	AllowedEmailDomains   []string
	BootstrapAdminEmails  []string
	SessionTTL            string
	SessionIdleTTL        string
	Validation            iam.OIDCValidationMetadataInput
	Activation            iam.OIDCActivationMetadataInput
}

type Repository interface {
	Get(ctx context.Context) (*StoredConfigRecord, error)
	Upsert(ctx context.Context, record StoredConfigRecord) (StoredConfigRecord, error)
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) Get(ctx context.Context) (*StoredConfigRecord, error) {
	item, err := r.client.InstanceAuthConfig.Query().
		Where(entinstanceauthconfig.ScopeKeyEQ(singletonScopeKey)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load instance auth config: %w", err)
	}
	record := mapStoredConfigRecord(item)
	return &record, nil
}

func (r *EntRepository) Upsert(ctx context.Context, record StoredConfigRecord) (StoredConfigRecord, error) {
	createBuilder := r.client.InstanceAuthConfig.Create().
		SetScopeKey(singletonScopeKey).
		SetStatus(record.Status).
		SetIssuerURL(record.IssuerURL).
		SetClientID(record.ClientID).
		SetRedirectMode(record.RedirectMode).
		SetRedirectURL(record.FixedRedirectURL).
		SetScopes(record.Scopes).
		SetEmailClaim(record.EmailClaim).
		SetNameClaim(record.NameClaim).
		SetUsernameClaim(record.UsernameClaim).
		SetGroupsClaim(record.GroupsClaim).
		SetAllowedEmailDomains(record.AllowedEmailDomains).
		SetBootstrapAdminEmails(record.BootstrapAdminEmails).
		SetSessionTTL(record.SessionTTL).
		SetSessionIdleTTL(record.SessionIdleTTL).
		SetValidationMetadata(iam.OIDCValidationMetadata{
			Status:                record.Validation.Status,
			Message:               record.Validation.Message,
			CheckedAt:             record.Validation.CheckedAt,
			IssuerURL:             record.Validation.IssuerURL,
			AuthorizationEndpoint: record.Validation.AuthorizationEndpoint,
			TokenEndpoint:         record.Validation.TokenEndpoint,
			RedirectURL:           record.Validation.RedirectURL,
			Warnings:              cloneStrings(record.Validation.Warnings),
		}).
		SetActivationMetadata(iam.OIDCActivationMetadata{
			ActivatedAt: record.Activation.ActivatedAt,
			ActivatedBy: record.Activation.ActivatedBy,
			Source:      record.Activation.Source,
			Message:     record.Activation.Message,
		})
	if record.ClientSecretEncrypted != nil {
		createBuilder = createBuilder.SetClientSecretEncrypted(record.ClientSecretEncrypted)
	}
	existing, err := r.client.InstanceAuthConfig.Query().
		Where(entinstanceauthconfig.ScopeKeyEQ(singletonScopeKey)).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		created, createErr := createBuilder.Save(ctx)
		if createErr != nil {
			return StoredConfigRecord{}, fmt.Errorf("create instance auth config: %w", createErr)
		}
		return mapStoredConfigRecord(created), nil
	case err != nil:
		return StoredConfigRecord{}, fmt.Errorf("query instance auth config: %w", err)
	}
	updateBuilder := r.client.InstanceAuthConfig.UpdateOneID(existing.ID).
		SetStatus(record.Status).
		SetIssuerURL(record.IssuerURL).
		SetClientID(record.ClientID).
		SetRedirectMode(record.RedirectMode).
		SetRedirectURL(record.FixedRedirectURL).
		SetScopes(record.Scopes).
		SetEmailClaim(record.EmailClaim).
		SetNameClaim(record.NameClaim).
		SetUsernameClaim(record.UsernameClaim).
		SetGroupsClaim(record.GroupsClaim).
		SetAllowedEmailDomains(record.AllowedEmailDomains).
		SetBootstrapAdminEmails(record.BootstrapAdminEmails).
		SetSessionTTL(record.SessionTTL).
		SetSessionIdleTTL(record.SessionIdleTTL).
		SetValidationMetadata(iam.OIDCValidationMetadata{
			Status:                record.Validation.Status,
			Message:               record.Validation.Message,
			CheckedAt:             record.Validation.CheckedAt,
			IssuerURL:             record.Validation.IssuerURL,
			AuthorizationEndpoint: record.Validation.AuthorizationEndpoint,
			TokenEndpoint:         record.Validation.TokenEndpoint,
			RedirectURL:           record.Validation.RedirectURL,
			Warnings:              cloneStrings(record.Validation.Warnings),
		}).
		SetActivationMetadata(iam.OIDCActivationMetadata{
			ActivatedAt: record.Activation.ActivatedAt,
			ActivatedBy: record.Activation.ActivatedBy,
			Source:      record.Activation.Source,
			Message:     record.Activation.Message,
		})
	if record.ClientSecretEncrypted != nil {
		updateBuilder = updateBuilder.SetClientSecretEncrypted(record.ClientSecretEncrypted)
	} else {
		updateBuilder = updateBuilder.ClearClientSecretEncrypted()
	}
	stored, err := updateBuilder.Save(ctx)
	if err != nil {
		return StoredConfigRecord{}, fmt.Errorf("update instance auth config: %w", err)
	}
	return mapStoredConfigRecord(stored), nil
}

func mapStoredConfigRecord(item *ent.InstanceAuthConfig) StoredConfigRecord {
	return StoredConfigRecord{
		Status:                item.Status,
		IssuerURL:             item.IssuerURL,
		ClientID:              item.ClientID,
		ClientSecretEncrypted: cloneSecret(item.ClientSecretEncrypted),
		RedirectMode:          item.RedirectMode,
		FixedRedirectURL:      item.RedirectURL,
		Scopes:                cloneStrings(item.Scopes),
		EmailClaim:            item.EmailClaim,
		NameClaim:             item.NameClaim,
		UsernameClaim:         item.UsernameClaim,
		GroupsClaim:           item.GroupsClaim,
		AllowedEmailDomains:   cloneStrings(item.AllowedEmailDomains),
		BootstrapAdminEmails:  cloneStrings(item.BootstrapAdminEmails),
		SessionTTL:            item.SessionTTL,
		SessionIdleTTL:        item.SessionIdleTTL,
		Validation: iam.OIDCValidationMetadataInput{
			Status:                item.ValidationMetadata.Status,
			Message:               item.ValidationMetadata.Message,
			CheckedAt:             item.ValidationMetadata.CheckedAt,
			IssuerURL:             item.ValidationMetadata.IssuerURL,
			AuthorizationEndpoint: item.ValidationMetadata.AuthorizationEndpoint,
			TokenEndpoint:         item.ValidationMetadata.TokenEndpoint,
			RedirectURL:           item.ValidationMetadata.RedirectURL,
			Warnings:              cloneStrings(item.ValidationMetadata.Warnings),
		},
		Activation: iam.OIDCActivationMetadataInput{
			ActivatedAt: item.ActivationMetadata.ActivatedAt,
			ActivatedBy: item.ActivationMetadata.ActivatedBy,
			Source:      item.ActivationMetadata.Source,
			Message:     item.ActivationMetadata.Message,
		},
	}
}

func cloneSecret(raw *iam.EncryptedSecret) *iam.EncryptedSecret {
	if raw == nil {
		return nil
	}
	cloned := *raw
	return &cloned
}

func cloneStrings(raw []string) []string {
	return append([]string(nil), raw...)
}
