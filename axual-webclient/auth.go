package webclient

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Global variables to cache the token source
var (
	tokenSourceCache oauth2.TokenSource
	tokenSourceMu    sync.Mutex
)

// tokenSourceFunc is a helper type that implements oauth2.TokenSource.
type tokenSourceFunc func() (*oauth2.Token, error)

func (f tokenSourceFunc) Token() (*oauth2.Token, error) {
	return f()
}

// contains checks whether a given slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getCachedTokenSource returns a singleton oauth2.TokenSource for Auth0 authentication.
func getCachedTokenSource(auth AuthStruct) oauth2.TokenSource {
	tokenSourceMu.Lock()
	defer tokenSourceMu.Unlock()

	// Create the token source only once.
	if tokenSourceCache == nil {
		// For long-lived sessions, we ensure offline_access is included.
		if auth.AuthMode == "auth0" && !contains(auth.Scopes, "offline_access") {
			auth.Scopes = append(auth.Scopes, "offline_access")
		}
		tokenSourceCache = oauth2.ReuseTokenSource(nil, tokenSourceFunc(func() (*oauth2.Token, error) {
			return getTokenWithAudience(auth)
		}))
	}
	return tokenSourceCache
}

// SignIn creates an authenticated HTTP client.
// For Auth0, it uses the cached token source.
// For Keycloak, it uses the normal password grant flow.
func SignIn(auth AuthStruct) (*http.Client, error) {
	switch auth.AuthMode {
	case "auth0":
		ts := getCachedTokenSource(auth)
		return oauth2.NewClient(context.Background(), ts), nil
	case "keycloak":
		userName := auth.Username
		password := auth.Password
		clientId := auth.ClientId
		conf := oauth2.Config{
			ClientID: clientId,
			Endpoint: oauth2.Endpoint{
				TokenURL: auth.Url,
			},
		}
		if auth.Scopes != nil {
			conf.Scopes = auth.Scopes
		}
		token, err := conf.PasswordCredentialsToken(context.Background(), userName, password)
		if err != nil {
			return nil, err
		}
		ts := oauth2.ReuseTokenSource(token, conf.TokenSource(context.Background(), token))
		return oauth2.NewClient(context.Background(), ts), nil
	default:
		return nil, fmt.Errorf("invalid auth mode: %s", auth.AuthMode)
	}
}

// getTokenWithAudience fetches an access token from Auth0.
// It adds the offline_access scope if missing to ensure a refresh token is issued.
func getTokenWithAudience(auth AuthStruct) (*oauth2.Token, error) {
	// Ensure offline_access is requested for Auth0.
	if auth.AuthMode == "auth0" && !contains(auth.Scopes, "offline_access") {
		auth.Scopes = append(auth.Scopes, "offline_access")
	}

	// Prepare the form data including the audience parameter.
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", auth.Username)
	data.Set("password", auth.Password)
	data.Set("client_id", auth.ClientId)
	data.Set("audience", auth.Audience)
	if len(auth.Scopes) > 0 {
		data.Set("scope", strings.Join(auth.Scopes, " "))
	}

	req, err := http.NewRequest("POST", auth.Url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get token, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		RefreshToken string `json:"refresh_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}

	// Build an oauth2.Token from the response, including the refresh token.
	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		RefreshToken: tokenResp.RefreshToken,
	}
	return token, nil
}
