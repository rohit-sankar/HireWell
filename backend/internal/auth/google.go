package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// googleEndpoint is equivalent to golang.org/x/oauth2/google.Endpoint. It's
// inlined here rather than importing the google subpackage because that
// subpackage transitively pulls in cloud.google.com/go/compute/metadata
// (for GCE metadata-server credential detection, which this application
// never uses -- we only ever do the standard web OAuth code exchange
// flow), and that dependency is unreachable in network environments that
// don't allowlist cloud.google.com. These are simply Google's published,
// stable OAuth2 endpoint URLs, not something that needs a library.
var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// NewGoogleOAuthConfig builds the oauth2.Config used for the
// /auth/google -> /auth/google/callback flow. Returns nil if Google OAuth
// is not configured (dev environments without a Google Cloud project can
// still run the rest of the app -- this feature degrades gracefully rather
// than crashing startup).
func NewGoogleOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	if clientID == "" || clientSecret == "" {
		return nil
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     googleEndpoint,
	}
}

// ExchangeGoogleCode swaps an OAuth authorization code for the user's
// verified profile info via Google's userinfo endpoint.
func ExchangeGoogleCode(ctx context.Context, cfg *oauth2.Config, code string) (*GoogleUserInfo, error) {
	if cfg == nil {
		return nil, errors.New("google oauth is not configured")
	}
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("google userinfo request failed")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if !info.EmailVerified {
		return nil, errors.New("google account email is not verified")
	}
	return &info, nil
}
