package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	iam "github.com/BetterAndBetterII/openase/internal/domain/iam"
)

type InstanceAuthConfig struct {
	ent.Schema
}

func (InstanceAuthConfig) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("scope_key").Default("instance").Immutable(),
		field.String("status").Default(string(iam.AccessControlStatusAbsent)),
		field.String("issuer_url").Default(""),
		field.String("client_id").Default(""),
		field.JSON("client_secret_encrypted", &iam.EncryptedSecret{}).Optional(),
		field.String("redirect_mode").Default(""),
		field.String("redirect_url").Default(""),
		textArrayField("scopes"),
		field.String("email_claim").Default("email"),
		field.String("name_claim").Default("name"),
		field.String("username_claim").Default("preferred_username"),
		field.String("groups_claim").Default("groups"),
		textArrayField("allowed_email_domains"),
		textArrayField("bootstrap_admin_emails"),
		field.String("session_ttl").Default("8h"),
		field.String("session_idle_ttl").Default("30m"),
		field.JSON("validation_metadata", iam.OIDCValidationMetadata{}).
			Default(func() iam.OIDCValidationMetadata { return iam.OIDCValidationMetadata{} }),
		field.JSON("activation_metadata", iam.OIDCActivationMetadata{}).
			Default(func() iam.OIDCActivationMetadata { return iam.OIDCActivationMetadata{} }),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (InstanceAuthConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope_key").Unique(),
	}
}
