package youtube

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/edward-/publish-social-network/internal/config"
	"github.com/edward-/publish-social-network/internal/domain"
	"github.com/edward-/publish-social-network/pkg/media"
)

// Publisher implements domain.Publisher for YouTube.
type Publisher struct {
	client    *Client
	mediaUtil *media.Validator
}

// NewPublisher creates a new YouTube publisher with injected configuration.
func NewPublisher(cfg config.YouTubeConfig) (*Publisher, error) {
	client, err := NewClient(cfg.ClientID, cfg.ClientSecret, cfg.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube client: %w", err)
	}

	return &Publisher{
		client:    client,
		mediaUtil: media.NewValidator(),
	}, nil
}

// Publish publishes a video post to YouTube.
func (p *Publisher) Publish(ctx context.Context, post domain.Post) (string, error) {
	// YouTube only supports video content
	if post.MediaType != domain.MediaTypeVideo {
		return "", fmt.Errorf("%w: YouTube only supports video content", domain.ErrUnsupportedMediaType)
	}

	// Validate title is present
	if post.Title == "" {
		return "", fmt.Errorf("title is required for YouTube posts")
	}

	// Read and validate the video file
	mediaInfo, reader, err := p.mediaUtil.ReadAndValidate(post.MediaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read media: %w", err)
	}
	defer reader.Close()

	// Read all video data
	videoData, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read video data: %w", err)
	}

	// Build description with caption and tags
	description := post.Caption
	if len(post.Tags) > 0 {
		if description != "" {
			description += "\n\n"
		}
		description += formatTagsAsHashtags(post.Tags)
	}

	// Default to "private" if not specified
	privacyStatus := "private"

	// Use goroutine for the upload
	type result struct {
		url string
		err error
	}
	resultCh := make(chan result, 1)

	go func() {
		url, err := p.client.UploadVideo(
			post.Title,
			description,
			post.Tags,
			privacyStatus,
			strings.NewReader(string(videoData)),
			mediaInfo.Size,
		)
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

// Platform returns the platform identifier.
func (p *Publisher) Platform() domain.Platform {
	return domain.YouTube
}

// ValidateConfig checks that the YouTube configuration is valid.
func (p *Publisher) ValidateConfig() error {
	if p.client.clientID == "" {
		return fmt.Errorf("YouTube ClientID is required")
	}
	if p.client.clientSecret == "" {
		return fmt.Errorf("YouTube ClientSecret is required")
	}
	if p.client.refreshToken == "" {
		return fmt.Errorf("YouTube RefreshToken is required")
	}
	return nil
}

// formatTagsAsHashtags formats tags as YouTube hashtags in description.
func formatTagsAsHashtags(tags []string) string {
	var result []string
	for _, tag := range tags {
		result = append(result, "#"+tag)
	}
	return strings.Join(result, " ")
}

// SetTimeout sets a custom timeout for the publisher.
func (p *Publisher) SetTimeout(timeout time.Duration) {
	// Note: The underlying HTTP client timeout can't be easily changed
	// after creation. This is a placeholder for future enhancement.
}
