package webclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// recorded is one observed request to the fake token endpoint.
type recorded struct {
	form       url.Values
	authHeader string
}

// tokenServer is a fake OAuth2 token endpoint that records every request.
//
// expiresIn is returned verbatim. A negative value is the trick that makes refresh
// testable: the issued token is already past its expiry, so the next Token call is a
// fresh exchange — no sleeping and no clock faking.
func tokenServer(t *testing.T, expiresIn int) (*httptest.Server, func() []recorded) {
	t.Helper()

	var mu sync.Mutex
	var seen []recorded

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}
		mu.Lock()
		seen = append(seen, recorded{form: r.PostForm, authHeader: r.Header.Get("Authorization")})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"at-%d","token_type":"Bearer","expires_in":%d}`, len(seen), expiresIn)
	}))
	t.Cleanup(srv.Close)

	return srv, func() []recorded {
		mu.Lock()
		defer mu.Unlock()
		return append([]recorded(nil), seen...)
	}
}

// failingTokenServer answers 200 for the first n requests and 401 after that.
func failingTokenServer(t *testing.T, okCount, expiresIn int) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	count := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		n := count
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if n > okCount {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_client","error_description":"Invalid client or Invalid client credentials"}`)
			return
		}
		fmt.Fprintf(w, `{"access_token":"at","token_type":"Bearer","expires_in":%d}`, expiresIn)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func formKeys(v url.Values) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestSecretTokenRequestBody(t *testing.T) {
	srv, requests := tokenServer(t, 3600)

	_, err := SignIn(Credentials{
		Mode:         ModeSecret,
		TokenURL:     srv.URL,
		ClientID:     "sa-orders-a1b2c3",
		ClientSecret: "shh",
	})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	got := requests()
	if len(got) != 1 {
		t.Fatalf("made %d token requests, want exactly 1", len(got))
	}

	want := []string{"client_id", "client_secret", "grant_type"}
	if keys := formKeys(got[0].form); !equal(keys, want) {
		t.Errorf("form keys = %v, want %v", keys, want)
	}
	if g := got[0].form.Get("grant_type"); g != "client_credentials" {
		t.Errorf("grant_type = %q", g)
	}
	if g := got[0].form.Get("client_id"); g != "sa-orders-a1b2c3" {
		t.Errorf("client_id = %q", g)
	}
	if g := got[0].form.Get("client_secret"); g != "shh" {
		t.Errorf("client_secret = %q", g)
	}
	// An Authorization header would mean the library probed HTTP Basic first, sending
	// the credentials somewhere we never intended.
	if got[0].authHeader != "" {
		t.Errorf("Authorization header = %q, want none", got[0].authHeader)
	}
}

func TestFederatedTokenRequestBody(t *testing.T) {
	srv, requests := tokenServer(t, 3600)

	_, err := SignIn(Credentials{
		Mode:      ModeFederated,
		TokenURL:  srv.URL,
		Assertion: "the-entra-token",
	})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	got := requests()
	if len(got) != 1 {
		t.Fatalf("made %d token requests, want exactly 1", len(got))
	}

	want := []string{"client_assertion", "client_assertion_type", "grant_type"}
	if keys := formKeys(got[0].form); !equal(keys, want) {
		t.Fatalf("form keys = %v, want exactly %v", keys, want)
	}
	if g := got[0].form.Get("grant_type"); g != "client_credentials" {
		t.Errorf("grant_type = %q", g)
	}
	if g := got[0].form.Get("client_assertion_type"); g != assertionType {
		t.Errorf("client_assertion_type = %q, want %q", g, assertionType)
	}
	if g := got[0].form.Get("client_assertion"); g != "the-entra-token" {
		t.Errorf("client_assertion = %q", g)
	}
	if got[0].authHeader != "" {
		t.Errorf("Authorization header = %q, want none", got[0].authHeader)
	}
}

func TestEagerFetchPrimesCache(t *testing.T) {
	srv, requests := tokenServer(t, 3600)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer at-1" {
			t.Errorf("API call carried %q, want the token from the eager fetch", got)
		}
	}))
	t.Cleanup(api.Close)

	client, err := SignIn(Credentials{
		Mode: ModeSecret, TokenURL: srv.URL, ClientID: "sa", ClientSecret: "shh",
	})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	for i := 0; i < 5; i++ {
		resp, err := client.Get(api.URL)
		if err != nil {
			t.Fatalf("API call %d: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	// If the eager fetch were wrapped a second time its token would be discarded and
	// the first API call would trigger another exchange.
	if n := len(requests()); n != 1 {
		t.Errorf("made %d token requests across 5 API calls, want 1", n)
	}
}

func TestEagerFetchFailsFast(t *testing.T) {
	srv := failingTokenServer(t, 0, 3600)

	_, err := SignIn(Credentials{
		Mode: ModeSecret, TokenURL: srv.URL, ClientID: "sa", ClientSecret: "the-secret-value",
	})
	if err == nil {
		t.Fatal("SignIn succeeded against a token endpoint that rejects everything")
	}
	if !strings.Contains(err.Error(), "client_id and client_secret") {
		t.Errorf("error should say what to check, got: %v", err)
	}
	if strings.Contains(err.Error(), "the-secret-value") {
		t.Errorf("error leaked the client secret: %v", err)
	}
}

func TestFederatedFailureNamesEveryCause(t *testing.T) {
	srv := failingTokenServer(t, 0, 3600)

	_, err := SignIn(Credentials{
		Mode: ModeFederated, TokenURL: srv.URL, Assertion: "the-assertion-value",
	})
	if err == nil {
		t.Fatal("SignIn succeeded against a token endpoint that rejects everything")
	}
	// Keycloak answers invalid_client for all of these, so the message has to list them.
	for _, cause := range []string{"expired", "subject", "issuer", "client-auth-federated", "federated-jwt"} {
		if !strings.Contains(err.Error(), cause) {
			t.Errorf("error does not mention %q: %v", cause, err)
		}
	}
	if strings.Contains(err.Error(), "the-assertion-value") {
		t.Errorf("error leaked the assertion: %v", err)
	}
}

func TestRefreshFailureMessageDiffers(t *testing.T) {
	// First exchange succeeds, every later one fails. expires_in is negative so the
	// token is stale immediately and the next call is a real exchange.
	srv := failingTokenServer(t, 1, -5)

	source, err := newTokenSource(context.Background(), Credentials{
		Mode: ModeFederated, TokenURL: srv.URL, Assertion: "tok",
	})
	if err != nil {
		t.Fatalf("first exchange should succeed: %v", err)
	}

	_, err = source.Token()
	if err == nil {
		t.Fatal("second exchange should fail")
	}
	if !strings.Contains(err.Error(), "expired") || !strings.Contains(err.Error(), "fedClientAssertionMaxExp") {
		t.Errorf("refresh failure should point at an expired assertion, got: %v", err)
	}
	// The four-causes message is for a configuration that never worked. Saying it here
	// would send the reader after prerequisites that are demonstrably fine.
	if strings.Contains(err.Error(), "client-auth-federated") {
		t.Errorf("refresh failure should not blame the platform prerequisites: %v", err)
	}
}

func TestOIDCTokenFileRereadOnRefresh(t *testing.T) {
	srv, requests := tokenServer(t, -5)

	path := filepath.Join(t.TempDir(), "assertion")
	if err := os.WriteFile(path, []byte("assertion-A"), 0o600); err != nil {
		t.Fatal(err)
	}

	source, err := newTokenSource(context.Background(), Credentials{
		Mode: ModeFederated, TokenURL: srv.URL, AssertionFile: path,
	})
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	if err := os.WriteFile(path, []byte("assertion-B"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("second exchange: %v", err)
	}

	got := requests()
	if len(got) != 2 {
		t.Fatalf("made %d token requests, want 2", len(got))
	}
	if a := got[0].form.Get("client_assertion"); a != "assertion-A" {
		t.Errorf("first exchange sent %q, want assertion-A", a)
	}
	if b := got[1].form.Get("client_assertion"); b != "assertion-B" {
		t.Errorf("refresh sent %q, want assertion-B — the file was not re-read", b)
	}
}

func TestOIDCTokenFileTrimsWhitespace(t *testing.T) {
	srv, requests := tokenServer(t, 3600)

	path := filepath.Join(t.TempDir(), "assertion")
	// Kubernetes projected tokens and shell redirects both leave a trailing newline.
	if err := os.WriteFile(path, []byte("the-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := SignIn(Credentials{Mode: ModeFederated, TokenURL: srv.URL, AssertionFile: path}); err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	if a := requests()[0].form.Get("client_assertion"); a != "the-token" {
		t.Errorf("client_assertion = %q, want the newline trimmed", a)
	}
}

func TestOIDCTokenFileUnreadable(t *testing.T) {
	srv, _ := tokenServer(t, 3600)
	path := filepath.Join(t.TempDir(), "missing")

	_, err := SignIn(Credentials{Mode: ModeFederated, TokenURL: srv.URL, AssertionFile: path})
	if err == nil {
		t.Fatal("SignIn succeeded with no assertion file")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should name the path, got: %v", err)
	}
	if !strings.Contains(err.Error(), "oidc_token_file") {
		t.Errorf("error should name the attribute, got: %v", err)
	}
}

func TestOIDCTokenFileEmpty(t *testing.T) {
	srv, _ := tokenServer(t, 3600)
	path := filepath.Join(t.TempDir(), "assertion")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := SignIn(Credentials{Mode: ModeFederated, TokenURL: srv.URL, AssertionFile: path})
	if err == nil {
		t.Fatal("SignIn succeeded with an empty assertion file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should say the file is empty, got: %v", err)
	}
}

func TestSecretRefreshResendsClientCredentials(t *testing.T) {
	srv, requests := tokenServer(t, -5)

	source, err := newTokenSource(context.Background(), Credentials{
		Mode: ModeSecret, TokenURL: srv.URL, ClientID: "sa", ClientSecret: "shh",
	})
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	got := requests()
	if len(got) != 2 {
		t.Fatalf("made %d token requests, want 2", len(got))
	}
	// Client credentials has no refresh token; a renewal is another full exchange.
	want := []string{"client_id", "client_secret", "grant_type"}
	if keys := formKeys(got[1].form); !equal(keys, want) {
		t.Errorf("refresh form keys = %v, want %v", keys, want)
	}
}

func TestZeroExpiresInNeverRefreshes(t *testing.T) {
	// Documents a trap rather than fixing it: with no expires_in the token has a zero
	// expiry, which oauth2 treats as "never expires". Keycloak always sends one.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at","token_type":"Bearer"}`)
	}))
	t.Cleanup(srv.Close)

	source, err := newTokenSource(context.Background(), Credentials{
		Mode: ModeFederated, TokenURL: srv.URL, Assertion: "tok",
	})
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	token, err := source.Token()
	if err != nil {
		t.Fatalf("second Token call: %v", err)
	}
	if !token.Expiry.IsZero() {
		t.Errorf("expected a zero expiry, got %v", token.Expiry)
	}
}

func TestROPCTokenRequestBody(t *testing.T) {
	srv, requests := tokenServer(t, 3600)

	_, err := SignIn(Credentials{
		Mode:     ModeROPC,
		TokenURL: srv.URL,
		ClientID: "self-service",
		Username: "someone@example.com",
		Password: "pw",
		Scopes:   []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	got := requests()
	if len(got) != 1 {
		t.Fatalf("made %d token requests, want 1", len(got))
	}
	if g := got[0].form.Get("grant_type"); g != "password" {
		t.Errorf("grant_type = %q", g)
	}
	if g := got[0].form.Get("username"); g != "someone@example.com" {
		t.Errorf("username = %q", g)
	}
	if g := got[0].form.Get("client_id"); g != "self-service" {
		t.Errorf("client_id = %q", g)
	}
	if g := got[0].form.Get("scope"); g != "openid profile" {
		t.Errorf("scope = %q", g)
	}
}

func TestUsesOwnTransportNotDefault(t *testing.T) {
	// A TLS server with a self-signed certificate no root store trusts. Reaching it at
	// all proves our transport carries InsecureSkipVerify.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at","token_type":"Bearer","expires_in":3600}`)
	}))
	t.Cleanup(srv.Close)

	before := http.DefaultTransport.(*http.Transport).TLSClientConfig

	if _, err := SignIn(Credentials{
		Mode: ModeSecret, TokenURL: srv.URL, ClientID: "sa", ClientSecret: "shh",
	}); err != nil {
		t.Fatalf("SignIn over an untrusted certificate: %v", err)
	}

	// The whole point of the change: the process-wide transport is left alone.
	if after := http.DefaultTransport.(*http.Transport).TLSClientConfig; after != before {
		t.Errorf("http.DefaultTransport.TLSClientConfig was modified (%v -> %v)", before, after)
	}
}

func TestSignInPreservesProxySettings(t *testing.T) {
	// Cloning the default transport rather than building a fresh one is what keeps
	// Proxy: ProxyFromEnvironment, which customers behind HTTPS_PROXY depend on.
	srv, _ := tokenServer(t, 3600)

	client, err := SignIn(Credentials{
		Mode: ModeSecret, TokenURL: srv.URL, ClientID: "sa", ClientSecret: "shh",
	})
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	transport, ok := client.Transport.(*oauth2.Transport)
	if !ok {
		t.Fatalf("client.Transport is %T, want *oauth2.Transport", client.Transport)
	}
	base, ok := transport.Base.(*http.Transport)
	if !ok {
		t.Fatalf("transport.Base is %T, want *http.Transport — a nil base silently falls back to http.DefaultTransport", transport.Base)
	}
	if base.Proxy == nil {
		t.Error("base transport has no Proxy; customers behind HTTPS_PROXY would break")
	}
	if client.Timeout != requestTimeout {
		t.Errorf("client.Timeout = %v, want %v — oauth2.NewClient does not inherit it", client.Timeout, requestTimeout)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
