package webclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// assertionType is the client_assertion_type for RFC 7523 JWT client authentication.
const assertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

const requestTimeout = 30 * time.Second

// SignIn returns an HTTP client that carries a valid access token on every request and
// renews it as it expires.
//
// The first token is fetched here, eagerly, so bad credentials fail while Terraform is
// still configuring the provider rather than partway through an apply.
func SignIn(c Credentials) (*http.Client, error) {
	// Clone rather than build a fresh transport: the default carries
	// Proxy: ProxyFromEnvironment, and customers behind HTTPS_PROXY depend on it.
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// ponytail: InsecureSkipVerify is preserved from the previous implementation, which
	// set it on http.DefaultTransport and so disabled verification for the whole
	// process. Scoping it to our own transport is the fix in this change. Verifying
	// certificates by default is a separate, breaking change — it stops on-premise
	// installs with self-signed certificates from working and needs its own deprecation.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402

	httpClient := &http.Client{Transport: transport, Timeout: requestTimeout}

	// Deliberately not Terraform's context. The one passed to Configure comes from a
	// gRPC handler and is cancelled as soon as Configure returns, but a token source
	// keeps its context for the life of the provider — so using it would make the first
	// token renewal during an apply fail with "context canceled". Requests stay bounded
	// by the client timeout above.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	tokenSource, err := newTokenSource(ctx, c)
	if err != nil {
		return nil, err
	}

	// NewClient copies only the Transport out of the context's client, so the timeout
	// has to be set again on the result.
	client := oauth2.NewClient(ctx, tokenSource)
	client.Timeout = requestTimeout
	return client, nil
}

// newTokenSource builds the token source for the resolved mode and fetches the first
// token from it before returning.
func newTokenSource(ctx context.Context, c Credentials) (oauth2.TokenSource, error) {
	if c.Mode == ModeROPC {
		conf := oauth2.Config{
			ClientID: c.ClientID,
			Scopes:   c.Scopes,
			Endpoint: oauth2.Endpoint{TokenURL: c.TokenURL, AuthStyle: oauth2.AuthStyleInParams},
		}
		token, err := conf.PasswordCredentialsToken(ctx, c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("authenticating with username and password: %w", err)
		}
		return conf.TokenSource(ctx, token), nil
	}

	source := &axualTokenSource{ctx: ctx, mode: c.Mode, build: configBuilder(c)}

	// Order matters. Wrapping first means oauth2.NewClient's own
	// ReuseTokenSource(nil, ts) call hands back this very object rather than wrapping
	// it again, so the token fetched just below survives and is not re-fetched.
	reusable := oauth2.ReuseTokenSource(nil, source)
	if _, err := reusable.Token(); err != nil {
		return nil, err
	}
	return reusable, nil
}

// axualTokenSource exchanges a service account credential for an access token. It sits
// inside a ReuseTokenSource, so Token is called only when a new token is actually
// needed — which is what lets a federated assertion be re-read from disk as it rotates.
type axualTokenSource struct {
	ctx   context.Context
	mode  AuthMode
	build func() (clientcredentials.Config, error)

	// succeeded records that an earlier exchange worked, which is the only way to tell
	// a broken configuration from an assertion that has since expired — Keycloak
	// reports both as invalid_client.
	//
	// No mutex: Token only ever runs inside reuseTokenSource.Token, which holds its own
	// lock for the whole call.
	succeeded bool
}

func (s *axualTokenSource) Token() (*oauth2.Token, error) {
	conf, err := s.build()
	if err != nil {
		return nil, err
	}
	token, err := conf.Token(s.ctx)
	if err != nil {
		return nil, s.explain(err)
	}
	s.succeeded = true
	return token, nil
}

// configBuilder returns a function producing the token request for this mode. It is a
// function rather than a value because a federated assertion may live in a file that is
// rewritten as it rotates, and must therefore be read afresh on every exchange.
func configBuilder(c Credentials) func() (clientcredentials.Config, error) {
	if c.Mode == ModeSecret {
		// Body: grant_type, client_id, client_secret. No scope — Platform Manager's
		// service account clients do not use one.
		//
		// AuthStyleInParams is not optional. Left unset, the library probes HTTP Basic
		// first, so the credentials go out in a header we never intended and a failure
		// is retried, sending the secret twice.
		conf := clientcredentials.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			TokenURL:     c.TokenURL,
			AuthStyle:    oauth2.AuthStyleInParams,
		}
		return func() (clientcredentials.Config, error) { return conf, nil }
	}

	return func() (clientcredentials.Config, error) {
		assertion := c.Assertion
		if c.AssertionFile != "" {
			contents, err := os.ReadFile(c.AssertionFile)
			if err != nil {
				return clientcredentials.Config{}, fmt.Errorf(
					"reading the federated assertion from oidc_token_file %q: %w", c.AssertionFile, err)
			}
			// CI redirects and Kubernetes projected tokens both leave a trailing
			// newline, which makes the JWT invalid.
			assertion = strings.TrimSpace(string(contents))
			if assertion == "" {
				return clientcredentials.Config{}, fmt.Errorf(
					"the federated assertion file %q is empty", c.AssertionFile)
			}
		}

		// Body: grant_type, client_assertion_type, client_assertion — and nothing else.
		// Leaving ClientID and ClientSecret empty is what omits them: the library writes
		// those fields only when they are non-empty. Keycloak resolves the service
		// account from the assertion's issuer and subject instead.
		return clientcredentials.Config{
			TokenURL:  c.TokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
			EndpointParams: url.Values{
				"client_assertion_type": {assertionType},
				"client_assertion":      {assertion},
			},
		}, nil
	}
}

// explain turns Keycloak's answer into something actionable. It reports invalid_client
// for several unrelated problems, so the message depends on whether an exchange has
// already worked during this run.
func (s *axualTokenSource) explain(err error) error {
	switch {
	case s.mode == ModeFederated && s.succeeded:
		return fmt.Errorf(
			"renewing the access token failed, but the federated assertion was accepted "+
				"earlier in this run, so the configuration is sound and the assertion has "+
				"most likely expired. An assertion supplied in oidc_token is minted once and "+
				"cannot be renewed. Use oidc_token_file with something that rewrites the file, "+
				"or raise the realm's fedClientAssertionMaxExp to cover the whole run: %w", err)

	case s.mode == ModeFederated:
		return fmt.Errorf(
			"federated authentication failed. Keycloak reports the same error for several "+
				"causes, so check each one: the assertion has expired or is malformed; the "+
				"service account is not bound to the assertion's subject; the realm has no "+
				"enabled identity provider for the assertion's issuer; or the platform is "+
				"missing the federated client authentication prerequisites (the "+
				"client-auth-federated feature, and the federated-jwt execution in the realm's "+
				"client authentication flow). Your platform administrator can confirm the last "+
				"two: %w", err)

	case s.succeeded:
		return fmt.Errorf(
			"renewing the access token failed, but the client secret was accepted earlier in "+
				"this run — it may have been rotated or the service account removed while "+
				"Terraform was running: %w", err)

	default:
		return fmt.Errorf(
			"service account authentication failed. Check that client_id and client_secret "+
				"are correct and belong to this tenant, and that authurl points at the tenant's "+
				"realm: %w", err)
	}
}
