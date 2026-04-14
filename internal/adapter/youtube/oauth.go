// Package youtube provides YouTube Data API v3 adapter with OAuth support.
package youtube

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

// OAuthConfig holds OAuth configuration for YouTube Login.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// DefaultOAuthConfig creates an OAuth2 config with YouTube scopes.
func DefaultOAuthConfig(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{youtube.YoutubeUploadScope, youtube.YoutubeReadonlyScope},
		Endpoint:     google.Endpoint,
	}
}

// ExchangeCode exchanges an authorization code for a token.
func ExchangeCode(cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	return cfg.Exchange(context.Background(), code)
}

// AuthURL returns the URL for the OAuth authorization flow.
func AuthURL(cfg *oauth2.Config, state string) string {
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// TokenSourceFromToken creates a token source from a token.
func TokenSourceFromToken(cfg *oauth2.Config, t *oauth2.Token) oauth2.TokenSource {
	return cfg.TokenSource(context.Background(), t)
}

// NewClientWithToken creates a YouTube client using an existing token.
func NewClientWithToken(clientID, clientSecret, refreshToken string) (*Client, error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{youtube.YoutubeUploadScope, youtube.YoutubeReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-1 * time.Hour), // Force refresh
	}

	tokenSource := config.TokenSource(context.Background(), token)
	httpClient := oauth2.NewClient(context.Background(), tokenSource)

	service, err := youtube.New(httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return &Client{
		service:      service,
		refreshToken: refreshToken,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
	}, nil
}
