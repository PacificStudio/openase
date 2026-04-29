package controlplaneurl

import "testing"

func TestAPIBaseURLFromControlPlaneURLConvertsWebsocketConnectEndpoint(t *testing.T) {
	t.Parallel()

	apiURL, err := APIBaseURLFromControlPlaneURL("wss://control.example.com/api/v1/machines/connect", false)
	if err != nil {
		t.Fatalf("APIBaseURLFromControlPlaneURL(resource) error = %v", err)
	}
	if apiURL != "https://control.example.com/api/v1" {
		t.Fatalf("APIBaseURLFromControlPlaneURL(resource) = %q", apiURL)
	}

	platformURL, err := APIBaseURLFromControlPlaneURL("wss://control.example.com/api/v1/machines/connect", true)
	if err != nil {
		t.Fatalf("APIBaseURLFromControlPlaneURL(platform) error = %v", err)
	}
	if platformURL != "https://control.example.com/api/v1/platform" {
		t.Fatalf("APIBaseURLFromControlPlaneURL(platform) = %q", platformURL)
	}
}

func TestResolveControlPlaneURLPrefersConfiguredBaseURL(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://openase.example.com")
	resolved, err := ResolveControlPlaneURL("", "127.0.0.1", 19836)
	if err != nil {
		t.Fatalf("ResolveControlPlaneURL() error = %v", err)
	}
	if resolved != "https://openase.example.com" {
		t.Fatalf("ResolveControlPlaneURL() = %q", resolved)
	}
}
