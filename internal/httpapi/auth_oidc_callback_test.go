package httpapi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
)

func TestOIDCCallbackCreatesBrowserSessionFromVerifiedIDToken(t *testing.T) {
	t.Parallel()

	issuerServer := newSignedOIDCProviderServer(t, "openase", "admin@example.com")
	defer issuerServer.Close()

	fixture := newHumanAuthFixtureWithConfig(t, config.AuthConfig{
		Mode: config.AuthModeOIDC,
		OIDC: config.OIDCConfig{
			IssuerURL:            issuerServer.URL,
			ClientID:             "openase",
			ClientSecret:         "test-client-secret",
			RedirectURL:          "http://127.0.0.1:19836/api/v1/auth/oidc/callback",
			Scopes:               []string{"openid", "profile", "email"},
			AllowedEmailDomains:  []string{"example.com"},
			BootstrapAdminEmails: []string{"admin@example.com"},
			SessionTTL:           8 * time.Hour,
			SessionIdleTTL:       30 * time.Minute,
		},
	})

	startRec := fixture.request(t, http.MethodGet, "/api/v1/auth/oidc/start?return_to=/projects/alpha", nil)
	if startRec.Code != http.StatusFound {
		t.Fatalf("oidc start status = %d: %s", startRec.Code, startRec.Body.String())
	}

	startLocation, err := url.Parse(startRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse oidc start redirect: %v", err)
	}
	state := strings.TrimSpace(startLocation.Query().Get("state"))
	if state == "" {
		t.Fatal("oidc start redirect missing state")
	}

	startCookies := startRec.Result().Cookies()
	if len(startCookies) != 1 || startCookies[0].Name != oidcFlowCookieName {
		t.Fatalf("expected oidc flow cookie, got %#v", startCookies)
	}

	callbackReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/oidc/callback?code=test-code&state="+url.QueryEscape(state),
		http.NoBody,
	)
	callbackReq.Header.Set("User-Agent", "OIDCCallbackTest/1.0")
	callbackReq.Header.Set("X-Forwarded-For", "203.0.113.8")
	callbackReq.AddCookie(startCookies[0])
	callbackRec := httptest.NewRecorder()

	fixture.server.Handler().ServeHTTP(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusFound {
		t.Fatalf("oidc callback status = %d: %s", callbackRec.Code, callbackRec.Body.String())
	}
	if location := callbackRec.Header().Get("Location"); location != "/projects/alpha" {
		t.Fatalf("callback redirect = %q, want /projects/alpha", location)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range callbackRec.Result().Cookies() {
		if cookie.Name == humanSessionCookieName && strings.TrimSpace(cookie.Value) != "" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected human session cookie in %#v", callbackRec.Result().Cookies())
	}
	if sessionCookie.Expires.Year() != 9999 {
		t.Fatalf("session cookie expiry = %s, want year 9999", sessionCookie.Expires)
	}

	sessionRec := fixture.request(t, http.MethodGet, "/api/v1/auth/session", map[string]string{
		"Cookie":          humanSessionCookieName + "=" + sessionCookie.Value,
		"User-Agent":      "OIDCCallbackTest/1.0",
		"X-Forwarded-For": "203.0.113.8",
	})
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("auth session status = %d: %s", sessionRec.Code, sessionRec.Body.String())
	}

	var sessionPayload authSessionResponse
	decodeResponse(t, sessionRec, &sessionPayload)
	if !sessionPayload.Authenticated {
		t.Fatalf("authenticated = false, payload = %+v", sessionPayload)
	}
	if sessionPayload.AuthMode != "oidc" {
		t.Fatalf("auth_mode = %q, want oidc", sessionPayload.AuthMode)
	}
	if sessionPayload.User == nil {
		t.Fatal("expected authenticated user payload")
	}
	if sessionPayload.User.PrimaryEmail != "admin@example.com" {
		t.Fatalf("primary_email = %q, want admin@example.com", sessionPayload.User.PrimaryEmail)
	}
	if !slices.Contains(sessionPayload.Roles, "instance_admin") {
		t.Fatalf("roles = %#v, want bootstrap instance_admin", sessionPayload.Roles)
	}
}

type signedOIDCProviderServer struct {
	*httptest.Server
	audience string
	email    string
	key      *rsa.PrivateKey
}

func newSignedOIDCProviderServer(t *testing.T, audience string, email string) *signedOIDCProviderServer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	provider := &signedOIDCProviderServer{
		audience: audience,
		email:    email,
		key:      privateKey,
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + server.URL + `","authorization_endpoint":"` + server.URL + `/authorize","token_endpoint":"` + server.URL + `/token","jwks_uri":"` + server.URL + `/jwks","response_types_supported":["code"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}`))
		case "/authorize":
			w.WriteHeader(http.StatusOK)
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			idToken := provider.mustSignIDToken(t, server.URL)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token",
				"id_token":     idToken,
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{
					{
						"kty": "RSA",
						"kid": "test-key",
						"use": "sig",
						"alg": "RS256",
						"n":   base64.RawURLEncoding.EncodeToString(provider.key.N.Bytes()),
						"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(provider.key.E)).Bytes()),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	provider.Server = server
	return provider
}

func (p *signedOIDCProviderServer) mustSignIDToken(t *testing.T, issuer string) string {
	t.Helper()

	now := time.Now().UTC()
	headerJSON, err := json.Marshal(map[string]any{
		"alg": "RS256",
		"kid": "test-key",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}
	claimsJSON, err := json.Marshal(map[string]any{
		"iss":                issuer,
		"sub":                "subject-admin",
		"aud":                p.audience,
		"exp":                now.Add(5 * time.Minute).Unix(),
		"iat":                now.Unix(),
		"email":              p.email,
		"email_verified":     true,
		"name":               "Admin Example",
		"preferred_username": "admin",
	})
	if err != nil {
		t.Fatalf("marshal jwt claims: %v", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
