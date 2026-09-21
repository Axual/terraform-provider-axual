package webclient

import (
	"fmt"
	"strings"
)

// AuthMode is how the provider proves who it is.
type AuthMode string

const (
	// ModeSecret is a service account authenticating with a client id and secret.
	ModeSecret AuthMode = "secret"
	// ModeFederated is a service account authenticating with an externally issued
	// assertion and no secret at all. Keycloak resolves which client is calling from
	// the assertion's issuer and subject, so no client id is sent either.
	ModeFederated AuthMode = "federated"
	// ModeROPC is the OAuth2 password grant, using a human's credentials. Deprecated:
	// OAuth 2.1 removes the grant, and service accounts replace it.
	ModeROPC AuthMode = "ropc"
)

// Environment variable names. The rule is AXUAL_ plus the attribute name, uppercased.
// The two legacy names predate that rule and keep their AUTH_ infix; they die with ROPC.
const (
	envClientID      = "AXUAL_CLIENT_ID"
	envClientSecret  = "AXUAL_CLIENT_SECRET"
	envOIDCToken     = "AXUAL_OIDC_TOKEN"
	envOIDCTokenFile = "AXUAL_OIDC_TOKEN_FILE"
	envUsername      = "AXUAL_AUTH_USERNAME"
	envPassword      = "AXUAL_AUTH_PASSWORD"
)

// ProviderConfig is the provider block, flattened to plain strings. An empty string
// means the attribute was not set in the configuration.
//
// This type deliberately holds no Terraform values. It is the seam that lets the whole
// of Resolve be tested with ordinary table tests instead of a constructed tfsdk config.
type ProviderConfig struct {
	AuthURL string

	ClientID       string // client_id
	LegacyClientID string // clientid, deprecated
	ClientSecret   string // client_secret
	OIDCToken      string // oidc_token
	OIDCTokenFile  string // oidc_token_file
	Username       string // username, deprecated
	Password       string // password, deprecated
	AuthMode       string // authmode, deprecated and inert
	Scopes         []string
}

// Credentials is a resolved, ready-to-use set of credentials for exactly one mode.
type Credentials struct {
	Mode     AuthMode
	TokenURL string

	ClientID     string // secret and ropc
	ClientSecret string // secret only

	Assertion     string // federated: the assertion itself
	AssertionFile string // federated: a path, re-read on every token refresh

	Username string   // ropc only
	Password string   // ropc only
	Scopes   []string // ropc only; neither service account mode sends a scope

	// Sources maps an attribute name to where its value came from: "configuration",
	// or the name of the environment variable. It never holds a credential value —
	// it exists so the resolved mode can be logged without leaking anything.
	Sources map[string]string
}

// String redacts every secret. Credentials must never reach a log any other way.
func (c Credentials) String() string {
	return fmt.Sprintf("Credentials{Mode:%s TokenURL:%s ClientID:%s ClientSecret:%s Assertion:%s AssertionFile:%s Username:%s Password:%s}",
		c.Mode, c.TokenURL, c.ClientID,
		redact(c.ClientSecret), redact(c.Assertion), c.AssertionFile,
		c.Username, redact(c.Password),
	)
}

func redact(s string) string {
	if s == "" {
		return "(unset)"
	}
	return "(redacted)"
}

// Warning is a non-fatal problem for the caller to surface however it surfaces
// warnings. Resolve does not know about Terraform diagnostics.
type Warning struct {
	Summary string
	Detail  string
}

// Resolve works out which authentication mode to use from which credentials are
// present, in the configuration or in the environment, and returns the credentials for
// that mode. getenv is os.Getenv in production.
//
// Resolve is pure: it performs no I/O, reads no globals, and never touches the network.
//
// Note for reviewers: this is deliberately not expressed with the framework's
// providervalidator.Conflicting / ExactlyOneOf helpers, even though that dependency is
// already present. Those validators only ever see the configuration, and every rule
// here can be triggered entirely from the environment — so a validator would reject
// configurations that are perfectly valid and pass ones that are not.
func Resolve(cfg ProviderConfig, getenv func(string) string) (Credentials, []Warning, error) {
	var warnings []Warning
	sources := map[string]string{}

	// pick prefers the configuration and falls back to the environment, recording
	// where the value came from so the resolved mode can be logged.
	pick := func(attr, configValue, envVar string) string {
		if configValue != "" {
			sources[attr] = "configuration"
			return configValue
		}
		if envValue := getenv(envVar); envValue != "" {
			sources[attr] = envVar
			return envValue
		}
		return ""
	}

	if strings.EqualFold(cfg.AuthMode, "auth0") {
		return Credentials{}, nil, fmt.Errorf(
			`authmode "auth0" is not supported. Remove the authmode attribute; ` +
				`the authentication method is determined by which credentials are supplied`)
	}

	clientID := pick("client_id", cfg.ClientID, envClientID)
	clientSecret := pick("client_secret", cfg.ClientSecret, envClientSecret)
	oidcToken := pick("oidc_token", cfg.OIDCToken, envOIDCToken)
	oidcTokenFile := pick("oidc_token_file", cfg.OIDCTokenFile, envOIDCTokenFile)
	username := pick("username", cfg.Username, envUsername)
	password := pick("password", cfg.Password, envPassword)

	// clientid is superseded by client_id. Accept both, prefer client_id, and refuse to
	// guess when they disagree.
	if cfg.LegacyClientID != "" {
		switch {
		case clientID == "":
			clientID = cfg.LegacyClientID
			sources["client_id"] = "configuration (clientid)"
		case clientID != cfg.LegacyClientID:
			return Credentials{}, nil, fmt.Errorf(
				"clientid and client_id are both set to different values (%q and %q). "+
					"Remove clientid; it is deprecated and client_id replaces it",
				cfg.LegacyClientID, clientID)
		}
	}

	// The mode triggers are exactly these three. client_id is NOT one of them: it is
	// shared vocabulary between secret mode and ROPC, and every existing configuration
	// already sets clientid = "self-service". Treating a client id as a service account
	// credential would silently flip those users into a mode they did not ask for.
	hasSecret := clientSecret != ""
	hasAssertion := oidcToken != "" || oidcTokenFile != ""

	if hasSecret && hasAssertion {
		return Credentials{}, nil, fmt.Errorf(
			"conflicting service account credentials: %s selects secret authentication, "+
				"but %s selects federated authentication. Supply exactly one",
			describe(sources, "client_secret"), describeAssertion(sources, oidcToken))
	}
	if oidcToken != "" && oidcTokenFile != "" {
		return Credentials{}, nil, fmt.Errorf(
			"oidc_token and oidc_token_file are both set (%s and %s). Supply exactly one",
			describe(sources, "oidc_token"), describe(sources, "oidc_token_file"))
	}

	creds := Credentials{TokenURL: cfg.AuthURL, Sources: sources}

	switch {
	case hasSecret:
		if clientID == "" {
			return Credentials{}, nil, fmt.Errorf(
				"client_secret is set (%s) but client_id is not. Secret service account "+
					"authentication needs both. Set client_id or %s",
				describe(sources, "client_secret"), envClientID)
		}
		creds.Mode = ModeSecret
		creds.ClientID = clientID
		creds.ClientSecret = clientSecret

	case hasAssertion:
		creds.Mode = ModeFederated
		creds.Assertion = oidcToken
		creds.AssertionFile = oidcTokenFile
		if clientID != "" {
			warnings = append(warnings, Warning{
				Summary: "client_id is ignored in federated authentication",
				Detail: fmt.Sprintf(
					"A client id was supplied (%s), but federated service account authentication "+
						"sends no client id. Keycloak resolves the service account from the "+
						"assertion's issuer and subject.",
					describe(sources, "client_id")),
			})
		}

	default:
		creds.Mode = ModeROPC
		creds.ClientID = clientID
		creds.Username = username
		creds.Password = password
		creds.Scopes = cfg.Scopes

		if username == "" {
			return Credentials{}, nil, fmt.Errorf(
				"no credentials found. Set client_id and client_secret for a service account, "+
					"or oidc_token / oidc_token_file for a federated service account. "+
					"Username and password authentication is deprecated; if you are still using "+
					"it, set username or %s", envUsername)
		}
		if password == "" {
			return Credentials{}, nil, fmt.Errorf(
				"username is set (%s) but password is not. Set password or %s",
				describe(sources, "username"), envPassword)
		}
		if clientID == "" {
			return Credentials{}, nil, fmt.Errorf(
				"username and password authentication needs a client id. Set client_id or %s",
				envClientID)
		}

		warnings = append(warnings, Warning{
			Summary: "Username and password authentication is deprecated",
			Detail: "The provider authenticated as a person using the OAuth2 password grant. " +
				"Use a service account instead: set client_id and client_secret, or " +
				"oidc_token / oidc_token_file for a federated service account. " +
				"Password authentication will be removed in a future release.",
		})
	}

	// Leftover password-grant values lose quietly. They are ignored, never fatal: a
	// stale AXUAL_AUTH_PASSWORD left in a pipeline must not fail the apply of someone
	// who is doing exactly the migration we are asking for.
	if creds.Mode != ModeROPC {
		if leftovers := describeLeftovers(sources, username, password); leftovers != "" {
			warnings = append(warnings, Warning{
				Summary: "Ignoring username and password",
				Detail: fmt.Sprintf(
					"Service account credentials were supplied, so the provider authenticated as a "+
						"service account and ignored %s. Remove them once the migration is complete.",
					leftovers),
			})
		}
		if len(cfg.Scopes) > 0 {
			warnings = append(warnings, Warning{
				Summary: "Ignoring scopes",
				Detail: "Service account authentication does not send a scope, so the scopes " +
					"attribute was ignored. It still applies to the deprecated username and " +
					"password authentication.",
			})
		}
	}

	return creds, warnings, nil
}

// describe names where an attribute's value came from, for an error or warning.
func describe(sources map[string]string, attr string) string {
	switch src := sources[attr]; src {
	case "":
		return attr
	case "configuration", "configuration (clientid)":
		return fmt.Sprintf("%s in the %s", attr, src)
	default:
		return fmt.Sprintf("the %s environment variable", src)
	}
}

func describeAssertion(sources map[string]string, oidcToken string) string {
	if oidcToken != "" {
		return describe(sources, "oidc_token")
	}
	return describe(sources, "oidc_token_file")
}

func describeLeftovers(sources map[string]string, username, password string) string {
	var found []string
	if username != "" {
		found = append(found, describe(sources, "username"))
	}
	if password != "" {
		found = append(found, describe(sources, "password"))
	}
	switch len(found) {
	case 0:
		return ""
	case 1:
		return found[0]
	default:
		return strings.Join(found[:len(found)-1], ", ") + " and " + found[len(found)-1]
	}
}
