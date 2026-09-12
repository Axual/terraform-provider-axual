package webclient

import (
	"strings"
	"testing"
)

// env builds a getenv func from a map, so tests never touch the real environment and
// can run in parallel without leaking into each other.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolve(t *testing.T) {
	t.Parallel()

	const authURL = "https://keycloak.example/realms/acme/protocol/openid-connect/token"

	cases := []struct {
		name string
		cfg  ProviderConfig
		env  map[string]string

		wantMode AuthMode
		wantErr  string   // substring; empty means no error expected
		wantWarn []string // substrings, each must appear in some warning
		noWarn   []string // substrings that must appear in no warning
		check    func(t *testing.T, c Credentials)
	}{
		{
			name:     "secret mode from configuration",
			cfg:      ProviderConfig{AuthURL: authURL, ClientID: "sa-orders-a1b2c3", ClientSecret: "shh"},
			wantMode: ModeSecret,
			check: func(t *testing.T, c Credentials) {
				if c.ClientID != "sa-orders-a1b2c3" || c.ClientSecret != "shh" {
					t.Errorf("wrong credentials: %+v", c)
				}
				if c.Sources["client_secret"] != "configuration" {
					t.Errorf("client_secret source = %q, want configuration", c.Sources["client_secret"])
				}
			},
		},
		{
			name:     "secret mode from environment",
			cfg:      ProviderConfig{AuthURL: authURL},
			env:      map[string]string{envClientID: "sa-orders-a1b2c3", envClientSecret: "shh"},
			wantMode: ModeSecret,
			check: func(t *testing.T, c Credentials) {
				if c.Sources["client_secret"] != envClientSecret {
					t.Errorf("client_secret source = %q, want %s", c.Sources["client_secret"], envClientSecret)
				}
				if c.Sources["client_id"] != envClientID {
					t.Errorf("client_id source = %q, want %s", c.Sources["client_id"], envClientID)
				}
			},
		},
		{
			name:     "configuration beats environment",
			cfg:      ProviderConfig{AuthURL: authURL, ClientID: "from-config", ClientSecret: "config-secret"},
			env:      map[string]string{envClientID: "from-env", envClientSecret: "env-secret"},
			wantMode: ModeSecret,
			check: func(t *testing.T, c Credentials) {
				if c.ClientID != "from-config" || c.ClientSecret != "config-secret" {
					t.Errorf("environment won over configuration: %+v", c)
				}
			},
		},
		{
			name:     "federated from oidc_token",
			cfg:      ProviderConfig{AuthURL: authURL, OIDCToken: "eyJhbGc..."},
			wantMode: ModeFederated,
			check: func(t *testing.T, c Credentials) {
				if c.Assertion != "eyJhbGc..." {
					t.Errorf("Assertion = %q", c.Assertion)
				}
				if c.AssertionFile != "" {
					t.Errorf("AssertionFile should be empty, got %q", c.AssertionFile)
				}
			},
		},
		{
			name:     "federated from oidc_token_file",
			cfg:      ProviderConfig{AuthURL: authURL, OIDCTokenFile: "/var/run/secrets/token"},
			wantMode: ModeFederated,
			check: func(t *testing.T, c Credentials) {
				if c.AssertionFile != "/var/run/secrets/token" {
					t.Errorf("AssertionFile = %q", c.AssertionFile)
				}
				if c.Assertion != "" {
					t.Errorf("Assertion should be empty, got %q", c.Assertion)
				}
			},
		},
		{
			name:    "oidc_token and oidc_token_file conflict",
			cfg:     ProviderConfig{AuthURL: authURL, OIDCToken: "tok", OIDCTokenFile: "/path"},
			wantErr: "both set",
		},
		{
			name:    "client_secret and oidc_token conflict",
			cfg:     ProviderConfig{AuthURL: authURL, ClientID: "sa", ClientSecret: "shh", OIDCToken: "tok"},
			wantErr: "conflicting service account credentials",
		},
		{
			name:    "conflict across configuration and environment",
			cfg:     ProviderConfig{AuthURL: authURL, ClientID: "sa", ClientSecret: "shh"},
			env:     map[string]string{envOIDCTokenFile: "/var/run/secrets/token"},
			wantErr: "conflicting service account credentials",
		},
		{
			name:    "client_secret without a client id",
			cfg:     ProviderConfig{AuthURL: authURL, ClientSecret: "shh"},
			wantErr: "client_id is not",
		},

		// The two guards that matter most. Every existing configuration in the world
		// sets clientid = "self-service", and pipelines will start setting
		// AXUAL_CLIENT_ID. A client id must never imply service account mode.
		{
			name: "client_id with password credentials stays on the password grant",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "self-service",
				Username: "someone@example.com", Password: "pw",
			},
			wantMode: ModeROPC,
			wantWarn: []string{"deprecated"},
		},
		{
			name:     "AXUAL_CLIENT_ID with password credentials stays on the password grant",
			cfg:      ProviderConfig{AuthURL: authURL, Username: "someone@example.com", Password: "pw"},
			env:      map[string]string{envClientID: "self-service"},
			wantMode: ModeROPC,
			wantWarn: []string{"deprecated"},
		},

		{
			name: "leftover username and password in configuration lose quietly",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "sa-orders-a1b2c3", ClientSecret: "shh",
				Username: "someone@example.com", Password: "pw",
			},
			wantMode: ModeSecret,
			wantWarn: []string{"Ignoring username and password", "username", "password"},
		},
		{
			name:     "leftover AXUAL_AUTH_PASSWORD loses quietly and is named",
			cfg:      ProviderConfig{AuthURL: authURL, ClientID: "sa-orders-a1b2c3", ClientSecret: "shh"},
			env:      map[string]string{envPassword: "stale"},
			wantMode: ModeSecret,
			wantWarn: []string{"Ignoring username and password", envPassword},
		},
		{
			name: "scopes are ignored in service account mode",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "sa-orders-a1b2c3", ClientSecret: "shh",
				Scopes: []string{"openid", "profile"},
			},
			wantMode: ModeSecret,
			wantWarn: []string{"Ignoring scopes"},
			check: func(t *testing.T, c Credentials) {
				if len(c.Scopes) != 0 {
					t.Errorf("Scopes should not reach a service account mode, got %v", c.Scopes)
				}
			},
		},
		{
			name:     "client id is ignored in federated mode",
			cfg:      ProviderConfig{AuthURL: authURL, LegacyClientID: "self-service", OIDCToken: "tok"},
			wantMode: ModeFederated,
			wantWarn: []string{"client_id is ignored"},
		},
		{
			name: "password grant keeps its scopes",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "self-service",
				Username: "someone@example.com", Password: "pw",
				Scopes: []string{"openid", "profile", "email"},
			},
			wantMode: ModeROPC,
			noWarn:   []string{"Ignoring scopes"},
			check: func(t *testing.T, c Credentials) {
				if len(c.Scopes) != 3 {
					t.Errorf("Scopes = %v, want all three", c.Scopes)
				}
			},
		},
		{
			name:    "username without password",
			cfg:     ProviderConfig{AuthURL: authURL, ClientID: "self-service", Username: "someone@example.com"},
			wantErr: "password is not",
		},
		{
			name:    "password grant without a client id",
			cfg:     ProviderConfig{AuthURL: authURL, Username: "someone@example.com", Password: "pw"},
			wantErr: "needs a client id",
		},
		{
			name:    "nothing set at all",
			cfg:     ProviderConfig{AuthURL: authURL},
			wantErr: "no credentials found",
		},
		{
			name: "legacy clientid alone still works",
			cfg: ProviderConfig{
				AuthURL: authURL, LegacyClientID: "self-service",
				Username: "someone@example.com", Password: "pw",
			},
			wantMode: ModeROPC,
			check: func(t *testing.T, c Credentials) {
				if c.ClientID != "self-service" {
					t.Errorf("ClientID = %q, want self-service", c.ClientID)
				}
			},
		},
		{
			name: "clientid and client_id agreeing is fine",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "same", LegacyClientID: "same",
				Username: "someone@example.com", Password: "pw",
			},
			wantMode: ModeROPC,
		},
		{
			name: "clientid and client_id disagreeing is an error",
			cfg: ProviderConfig{
				AuthURL: authURL, ClientID: "one", LegacyClientID: "two",
				Username: "someone@example.com", Password: "pw",
			},
			wantErr: "both set to different values",
		},
		{
			name:    "authmode auth0 is rejected",
			cfg:     ProviderConfig{AuthURL: authURL, AuthMode: "auth0", ClientID: "sa", ClientSecret: "shh"},
			wantErr: "is not supported",
		},
		{
			name:     "authmode keycloak is accepted and inert",
			cfg:      ProviderConfig{AuthURL: authURL, AuthMode: "keycloak", ClientID: "sa", ClientSecret: "shh"},
			wantMode: ModeSecret,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			creds, warnings, err := Resolve(tc.cfg, env(tc.env))

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got none (mode %s)", tc.wantErr, creds.Mode)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if creds.Mode != tc.wantMode {
				t.Errorf("mode = %s, want %s", creds.Mode, tc.wantMode)
			}
			if creds.TokenURL != tc.cfg.AuthURL {
				t.Errorf("TokenURL = %q, want %q", creds.TokenURL, tc.cfg.AuthURL)
			}

			all := allWarnings(warnings)
			for _, want := range tc.wantWarn {
				if !strings.Contains(all, want) {
					t.Errorf("no warning contains %q; warnings were:\n%s", want, all)
				}
			}
			for _, unwanted := range tc.noWarn {
				if strings.Contains(all, unwanted) {
					t.Errorf("warning unexpectedly contains %q; warnings were:\n%s", unwanted, all)
				}
			}

			if tc.check != nil {
				tc.check(t, creds)
			}
		})
	}
}

func allWarnings(ws []Warning) string {
	var b strings.Builder
	for _, w := range ws {
		b.WriteString(w.Summary)
		b.WriteString(" | ")
		b.WriteString(w.Detail)
		b.WriteString("\n")
	}
	return b.String()
}

func TestCredentialsStringRedacts(t *testing.T) {
	t.Parallel()

	creds := Credentials{
		Mode:         ModeSecret,
		ClientID:     "sa-orders-a1b2c3",
		ClientSecret: "the-actual-secret",
		Assertion:    "the-actual-assertion",
		Password:     "the-actual-password",
	}

	got := creds.String()
	for _, secret := range []string{"the-actual-secret", "the-actual-assertion", "the-actual-password"} {
		if strings.Contains(got, secret) {
			t.Errorf("String() leaked %q:\n%s", secret, got)
		}
	}
	if !strings.Contains(got, "sa-orders-a1b2c3") {
		t.Errorf("String() should keep the client id for diagnosis:\n%s", got)
	}
}
