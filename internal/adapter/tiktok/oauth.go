// Package tiktok provides the TikTok Content Posting API adapter.
package tiktok

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authEndpoint  = "https://www.tiktok.com/v2/auth/authorize/"
	tokenEndpoint = "https://open.tiktokapis.com/v2/oauth/token/"
)

// OAuthConfig holds OAuth configuration for TikTok Login Kit.
type OAuthConfig struct {
	ClientKey    string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// TokenResponse represents the response from TikTok token endpoint.
type TokenResponse struct {
	Data struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		OpenID       string `json:"open_id"`
		Scope        string `json:"scope"`
		Error        struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

// RefreshTokenResponse represents the response from refreshing a token.
type RefreshTokenResponse struct {
	Data struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Error        struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

// generateCodeVerifier creates a random PKCE code verifier (43-128 chars).
// Uses only characters allowed by RFC 7636: A-Z, a-z, 0-9, -, ., _, ~
func generateCodeVerifier() (string, error) {
	// 32 bytes decodes to exactly 43 base64url characters (no padding)
	// This meets the 43-128 character requirement
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	// Use RawURLEncoding (no padding) to get 43 characters
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge creates a SHA256 PKCE code challenge from a verifier.
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	// Use RawURLEncoding (no padding) as required by RFC 7636
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GeneratePKCE generates a code verifier and code challenge pair for PKCE OAuth.
func GeneratePKCE() (verifier, challenge string, err error) {
	verifier, err = generateCodeVerifier()
	if err != nil {
		return "", "", err
	}
	challenge = generateCodeChallenge(verifier)
	return verifier, challenge, nil
}

// VerifyPKCE verifies the PKCE verifier and challenge match.
// Returns true if challenge = SHA256(verifier) in base64url encoding.
func VerifyPKCE(verifier, challenge string) bool {
	computed := generateCodeChallenge(verifier)
	return computed == challenge
}

// BuildAuthorizationURL builds the TikTok authorization URL with PKCE.
func BuildAuthorizationURL(cfg OAuthConfig, state, codeChallenge string) string {
	params := url.Values{}
	params.Set("client_key", cfg.ClientKey)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(cfg.Scopes, ","))
	params.Set("redirect_uri", cfg.RedirectURI)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	return authEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for access tokens.
func ExchangeCode(cfg OAuthConfig, code, codeVerifier string) (*TokenResponse, error) {
	fmt.Printf("DEBUG: ExchangeCode called with:\n")
	fmt.Printf("  client_key: %s\n", cfg.ClientKey)
	fmt.Printf("  redirect_uri: %s\n", cfg.RedirectURI)
	fmt.Printf("  code: %s\n", code)
	fmt.Printf("  code_verifier: %s\n", codeVerifier)

	// Verify PKCE locally
	computed := generateCodeChallenge(codeVerifier)
	fmt.Printf("  computed_challenge: %s\n", computed)
	fmt.Printf("  verifier_len: %d\n", len(codeVerifier))

	// Build form body - send code as-is (Go's http.PostForm will encode it)
	data := url.Values{}
	data.Set("client_key", cfg.ClientKey)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", cfg.RedirectURI)
	data.Set("code_verifier", codeVerifier)

	encoded := data.Encode()
	fmt.Printf("DEBUG: Full form body: %s\n", encoded)

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug: print raw response
	fmt.Printf("DEBUG: Raw token response: %s\n", string(body))

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Data.Error.Code != "" {
		return nil, fmt.Errorf("token exchange failed: %s (code %s)", tokenResp.Data.Error.Message, tokenResp.Data.Error.Code)
	}

	return &tokenResp, nil
}

// RefreshAccessToken refreshes an access token using a refresh token.
func RefreshAccessToken(cfg OAuthConfig, refreshToken string) (*RefreshTokenResponse, error) {
	data := url.Values{}
	data.Set("client_key", cfg.ClientKey)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var refreshResp RefreshTokenResponse
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if refreshResp.Data.Error.Code != "" {
		return nil, fmt.Errorf("token refresh failed: %s (code %s)", refreshResp.Data.Error.Message, refreshResp.Data.Error.Code)
	}

	return &refreshResp, nil
}

// TokenInfo holds parsed token information for storage.
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	OpenID       string
	Scope        string
}

// ParseTokenResponse converts a TokenResponse to TokenInfo.
func ParseTokenResponse(resp *TokenResponse) *TokenInfo {
	return &TokenInfo{
		AccessToken:  resp.Data.AccessToken,
		RefreshToken: resp.Data.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second),
		OpenID:       resp.Data.OpenID,
		Scope:        resp.Data.Scope,
	}
}

// ParseRefreshTokenResponse converts a RefreshTokenResponse to TokenInfo.
func ParseRefreshTokenResponse(resp *RefreshTokenResponse) *TokenInfo {
	return &TokenInfo{
		AccessToken:  resp.Data.AccessToken,
		RefreshToken: resp.Data.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second),
	}
}

// IsTokenExpiringSoon checks if a token will expire within the given duration.
func (t *TokenInfo) IsTokenExpiringSoon(within time.Duration) bool {
	return time.Now().Add(within).After(t.ExpiresAt)
}