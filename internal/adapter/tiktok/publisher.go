// Package tiktok provides the TikTok Content Posting API adapter.
package tiktok

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/edward-/publish-social-network/internal/config"
	"github.com/edward-/publish-social-network/internal/domain"
	"github.com/edward-/publish-social-network/pkg/media"
)

// Publisher implements domain.Publisher for TikTok.
type Publisher struct {
	client       *Client
	mediaUtil    *media.Validator
	refreshToken string
	clientKey    string
	mu           sync.RWMutex
}

// NewPublisher creates a new TikTok publisher with injected configuration.
func NewPublisher(cfg config.TikTokConfig) *Publisher {
	return &Publisher{
		client:       NewClientWithRefresh(cfg.AccessToken, cfg.RefreshToken, cfg.ClientKey),
		mediaUtil:    media.NewValidator(),
		refreshToken: cfg.RefreshToken,
		clientKey:    cfg.ClientKey,
	}
}

// Publish publishes a video post to TikTok.
// TikTok only supports video content. This implements the Content Posting API v2
// which requires: OAuth token with video.upload and video.publish scopes.
func (p *Publisher) Publish(ctx context.Context, post domain.Post) (string, error) {
	// TikTok only supports video
	if post.MediaType != domain.MediaTypeVideo {
		return "", fmt.Errorf("%w: TikTok only supports video content", domain.ErrUnsupportedMediaType)
	}

	// Check and refresh token if needed
	if err := p.refreshTokenIfNeeded(); err != nil {
		log.Printf("Warning: token refresh check failed: %v", err)
	}

	// Read and validate the video file
	_, reader, err := p.mediaUtil.ReadAndValidate(post.MediaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read media: %w", err)
	}
	defer reader.Close()

	// Read all video data
	videoData, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read video data: %w", err)
	}

	// Build title and description
	title := post.Title
	if title == "" {
		title = post.Caption
	}

	// Extract first 100 chars for description if caption is too long
	description := post.Caption
	if len(description) > 2200 {
		description = description[:2197] + "..."
	}

	// Convert tags to TikTok hashtag format
	tags := post.Tags
	if len(tags) == 0 {
		tags = []string{}
	}

	// Use goroutine for the potentially long upload
	type result struct {
		url string
		err error
	}
	resultCh := make(chan result, 1)

	go func() {
		url, err := p.client.UploadVideo(videoData, title, description, tags, "public")
		resultCh <- result{url: url, err: err}
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%w: video upload timed out", domain.ErrTimeout)
	case r := <-resultCh:
		if r.err != nil {
			return "", r.err
		}
		return r.url, nil
	}
}

// refreshTokenIfNeeded checks if the token needs refreshing and refreshes it.
// TikTok tokens typically expire, so we refresh when we have a refresh token
// and the access token is getting close to expiry.
func (p *Publisher) refreshTokenIfNeeded() error {
	p.mu.RLock()
	refreshToken := p.refreshToken
	clientKey := p.clientKey
	p.mu.RUnlock()

	if refreshToken == "" || clientKey == "" {
		return nil // No refresh token available
	}

	// For now, we just check if access token exists
	// In a production system, you would track token expiry and refresh proactively
	// TikTok tokens typically last 24 hours
	accessToken := p.client.GetAccessToken()
	if accessToken == "" {
		return fmt.Errorf("no access token available")
	}

	return nil
}

// RefreshToken manually refreshes the access token using the refresh token.
func (p *Publisher) RefreshToken() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	refreshToken := p.refreshToken
	clientKey := p.clientKey

	if refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	oauthCfg := OAuthConfig{
		ClientKey: clientKey,
	}

	refreshResp, err := RefreshAccessToken(oauthCfg, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	p.client.SetAccessToken(refreshResp.Data.AccessToken)
	p.refreshToken = refreshResp.Data.RefreshToken

	log.Println("TikTok access token refreshed successfully")
	return nil
}

// Platform returns the platform identifier.
func (p *Publisher) Platform() domain.Platform {
	return domain.TikTok
}

// ValidateConfig checks that the TikTok configuration is valid.
func (p *Publisher) ValidateConfig() error {
	if p.client.GetAccessToken() == "" {
		return fmt.Errorf("TikTok AccessToken is required")
	}
	if p.clientKey == "" {
		return fmt.Errorf("TikTok ClientKey is required")
	}
	return nil
}

// SetTimeout sets a custom timeout for the publisher.
func (p *Publisher) SetTimeout(timeout time.Duration) {
	p.client.httpClient.Timeout = timeout
}
