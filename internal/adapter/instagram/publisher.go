package instagram

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/edward-/publish-social-network/internal/config"
	"github.com/edward-/publish-social-network/internal/domain"
	"github.com/edward-/publish-social-network/pkg/media"
)

// Publisher implements domain.Publisher for Instagram.
type Publisher struct {
	client    *Client
	mediaUtil *media.Validator
	httpClient *http.Client
}

// NewPublisher creates a new Instagram publisher with injected configuration.
func NewPublisher(cfg config.InstagramConfig) *Publisher {
	return &Publisher{
		client:    NewClient(cfg.UserID, cfg.AccessToken),
		mediaUtil: media.NewValidator(),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Publish publishes a post to Instagram.
// Instagram requires a two-step process: create a media container, then publish it.
// For images, the image must be hosted on a public URL. This implementation
// demonstrates the API flow but requires a URL for image hosting.
func (p *Publisher) Publish(ctx context.Context, post domain.Post) (string, error) {
	switch post.MediaType {
	case domain.MediaTypeImage:
		return p.publishImage(ctx, post)

	case domain.MediaTypeVideo:
		return p.publishVideo(ctx, post)

	case domain.MediaTypeText:
		return p.publishText(ctx, post)

	default:
		return "", fmt.Errorf("%w: %s", domain.ErrUnsupportedMediaType, post.MediaType)
	}
}

// publishImage handles image posts to Instagram.
// Note: Instagram Graph API requires images to be hosted on a public URL.
// This implementation demonstrates the flow but in production would need
// a temporary image hosting step.
func (p *Publisher) publishImage(ctx context.Context, post domain.Post) (string, error) {
	// For Instagram, we need to either:
	// 1. Host the image on a public URL and use image_url parameter, or
	// 2. Upload to Facebook first and get the URL

	// Read local image as fallback - in production, upload to temp hosting
	_, reader, err := p.mediaUtil.ReadAndValidate(post.MediaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read media: %w", err)
	}
	defer reader.Close()

	// For demonstration, this would need a public URL in production
	// The actual implementation would upload to temp hosting first
	return "", fmt.Errorf("image upload requires public URL hosting - use URL-based upload or upload to temp hosting first")
}

// publishVideo handles video posts to Instagram (Reels).
func (p *Publisher) publishVideo(ctx context.Context, post domain.Post) (string, error) {
	// Read local video
	_, reader, err := p.mediaUtil.ReadAndValidate(post.MediaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read media: %w", err)
	}
	defer reader.Close()

	// For video uploads, Instagram requires the video to be hosted on a public URL
	return "", fmt.Errorf("video upload requires public URL hosting - upload to temp hosting first")
}

// publishText handles text-only posts to Instagram.
// Instagram doesn't support pure text posts like Facebook does,
// so we create an image with the text.
func (p *Publisher) publishText(ctx context.Context, post domain.Post) (string, error) {
	return "", fmt.Errorf("%w: Instagram requires media (image or video)", domain.ErrUnsupportedMediaType)
}

// Platform returns the platform identifier.
func (p *Publisher) Platform() domain.Platform {
	return domain.Instagram
}

// ValidateConfig checks that the Instagram configuration is valid.
func (p *Publisher) ValidateConfig() error {
	if p.client.userID == "" {
		return fmt.Errorf("Instagram UserID is required")
	}
	if p.client.accessToken == "" {
		return fmt.Errorf("Instagram AccessToken is required")
	}
	return nil
}

// formatTags formats tags as hashtags.
func formatTags(tags []string) string {
	result := ""
	for _, tag := range tags {
		result += "#" + tag + " "
	}
	return result
}

// uploadToTempHosting uploads a file to temporary hosting and returns the URL.
// This is a placeholder - in production, integrate with a cloud storage service.
func (p *Publisher) uploadToTempHosting(ctx context.Context, filename string, data []byte) (string, error) {
	// Placeholder for temp hosting upload
	// In production: upload to S3, GCS, or similar and return the public URL
	return "", fmt.Errorf("temp hosting not implemented - upload to a public URL and use URL-based upload")
}
