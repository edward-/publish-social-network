package facebook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/edward-/publish-social-network/internal/config"
	"github.com/edward-/publish-social-network/internal/domain"
	"github.com/edward-/publish-social-network/pkg/media"
)

// Publisher implements domain.Publisher for Facebook.
type Publisher struct {
	client    *Client
	mediaUtil *media.Validator
}

// NewPublisher creates a new Facebook publisher with injected configuration.
func NewPublisher(cfg config.FacebookConfig) *Publisher {
	return &Publisher{
		client:    NewClient(cfg.PageID, cfg.AccessToken),
		mediaUtil: media.NewValidator(),
	}
}

// Publish publishes a post to Facebook.
func (p *Publisher) Publish(ctx context.Context, post domain.Post) (string, error) {
	switch post.MediaType {
	case domain.MediaTypeText:
		return p.client.PublishFeed(post.Caption)

	case domain.MediaTypeImage:
		return p.publishImage(ctx, post)

	case domain.MediaTypeVideo:
		return "", fmt.Errorf("%w: Facebook does not support direct video upload via Graph API", domain.ErrUnsupportedMediaType)

	default:
		return "", fmt.Errorf("%w: %s", domain.ErrUnsupportedMediaType, post.MediaType)
	}
}

// publishImage uploads and publishes an image to Facebook.
func (p *Publisher) publishImage(ctx context.Context, post domain.Post) (string, error) {
	// Read and validate the image file
	_, reader, err := p.mediaUtil.ReadAndValidate(post.MediaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read media: %w", err)
	}
	defer reader.Close()

	// Read all image data
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Build caption with tags
	caption := post.Caption
	if len(post.Tags) > 0 {
		caption += "\n\n" + formatTags(post.Tags)
	}

	return p.client.PublishPhoto(caption, imageData)
}

// Platform returns the platform identifier.
func (p *Publisher) Platform() domain.Platform {
	return domain.Facebook
}

// ValidateConfig checks that the Facebook configuration is valid.
func (p *Publisher) ValidateConfig() error {
	if p.client.pageID == "" {
		return fmt.Errorf("Facebook PageID is required")
	}
	if p.client.accessToken == "" {
		return fmt.Errorf("Facebook AccessToken is required")
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

// multipartWriter is a helper for creating multipart form data.
type multipartWriter struct {
	writer *multipart.Writer
	body   *bytes.Buffer
}

// newMultipartWriter creates a new multipart writer helper.
func newMultipartWriter(body *bytes.Buffer) *multipartWriter {
	return &multipartWriter{
		writer: multipart.NewWriter(body),
	}
}

// WriteField writes a form field.
func (mw *multipartWriter) WriteField(field, value string) {
	mw.writer.WriteField(field, value)
}

// WriteFile writes a file field.
func (mw *multipartWriter) WriteFile(fieldname, filename, mimeType string, data []byte) {
	part, _ := mw.writer.CreateFormFile(fieldname, filename)
	part.Write(data)
}

// Close closes the multipart writer.
func (mw *multipartWriter) Close() error {
	return mw.writer.Close()
}

// FormDataContentType returns the content type for the form data.
func (mw *multipartWriter) FormDataContentType() string {
	return mw.writer.FormDataContentType()
}
