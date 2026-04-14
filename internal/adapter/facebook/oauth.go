// Package facebook provides the Facebook Graph API adapter.
package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authEndpoint = "https://www.facebook.com/v18.0/dialog/oauth"
	tokenEndpoint = "https://graph.facebook.com/v18.0/oauth/access_token"
)

// OAuthConfig holds OAuth configuration for Facebook Login.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// TokenResponse represents the response from Facebook token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Error        *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// BuildAuthorizationURL builds the Facebook authorization URL.
func BuildAuthorizationURL(cfg OAuthConfig, state string) string {
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURI)
	params.Set("scope", strings.Join(cfg.Scopes, ","))
	params.Set("response_type", "code")
	params.Set("state", state)

	return authEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for access token.
func ExchangeCode(cfg OAuthConfig, code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", cfg.RedirectURI)

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != nil {
		return nil, fmt.Errorf("token exchange failed: %s (code %d)", tokenResp.Error.Message, tokenResp.Error.Code)
	}

	return &tokenResp, nil
}

// TokenInfo holds parsed token information for storage.
type TokenInfo struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
}

// ParseTokenResponse converts a TokenResponse to TokenInfo.
func ParseTokenResponse(resp *TokenResponse) *TokenInfo {
	return &TokenInfo{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
	}
}
