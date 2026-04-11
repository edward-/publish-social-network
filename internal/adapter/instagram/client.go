// Package instagram provides the Instagram Graph API adapter.
package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/edward-/publish-social-network/internal/domain"
)

// Client is an Instagram Graph API client.
type Client struct {
	userID      string
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

// NewClient creates a new Instagram API client.
func NewClient(userID, accessToken string) *Client {
	return &Client{
		userID:      userID,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: "https://graph.facebook.com/v18.0",
	}
}

// MediaContainerResponse represents the response from creating a media container.
type MediaContainerResponse struct {
	ID string `json:"id"`
}

// MediaPublishResponse represents the response from publishing media.
type MediaPublishResponse struct {
	CreationID string `json:"creation_id"`
	Success   bool   `json:"success"`
}

// ErrorResponse represents an Instagram API error.
type ErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// CreateMediaContainer creates a media container for publishing.
// This is step 1 of the two-step Instagram publishing process.
func (c *Client) CreateMediaContainer(imageURL, caption string, isReel bool) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/media", c.baseURL, c.userID)

	values := url.Values{}
	values.Set("caption", caption)
	values.Set("access_token", c.accessToken)

	// For image URLs, use image_url parameter
	if !isReel {
		values.Set("image_url", imageURL)
	}

	resp, err := c.httpClient.PostForm(endpoint, values)
	if err != nil {
		return "", fmt.Errorf("failed to create media container: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", c.mapError(errResp.Error.Code, errResp.Error.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var containerResp MediaContainerResponse
	if err := json.Unmarshal(respBody, &containerResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return containerResp.ID, nil
}

// CreateVideoContainer creates a media container for video content.
func (c *Client) CreateVideoContainer(videoURL, caption string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/media", c.baseURL, c.userID)

	values := url.Values{}
	values.Set("caption", caption)
	values.Set("video_url", videoURL)
	values.Set("access_token", c.accessToken)

	resp, err := c.httpClient.PostForm(endpoint, values)
	if err != nil {
		return "", fmt.Errorf("failed to create video container: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", c.mapError(errResp.Error.Code, errResp.Error.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var containerResp MediaContainerResponse
	if err := json.Unmarshal(respBody, &containerResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return containerResp.ID, nil
}

// PublishMedia publishes a media container that was previously created.
func (c *Client) PublishMedia(creationID string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/media_publish", c.baseURL, c.userID)

	values := url.Values{}
	values.Set("creation_id", creationID)
	values.Set("access_token", c.accessToken)

	resp, err := c.httpClient.PostForm(endpoint, values)
	if err != nil {
		return "", fmt.Errorf("failed to publish media: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", c.mapError(errResp.Error.Code, errResp.Error.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var publishResp MediaPublishResponse
	if err := json.Unmarshal(respBody, &publishResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return fmt.Sprintf("https://www.instagram.com/p/%s/", publishResp.CreationID), nil
}

// UploadPhoto uploads a photo and publishes it to Instagram.
func (c *Client) UploadPhoto(imageData []byte, caption string) (string, error) {
	// Note: Instagram Graph API requires the image to be hosted on a public URL.
	// For local images, you would need to first upload to a temporary hosting service.
	// This is a simplified implementation that demonstrates the two-step process.

	// Step 1: Create media container
	containerID, err := c.CreateMediaContainer("", caption, false)
	if err != nil {
		return "", fmt.Errorf("failed to create media container: %w", err)
	}

	// Step 2: Publish the container
	return c.PublishMedia(containerID)
}

// mapError maps Instagram API errors to domain errors.
func (c *Client) mapError(apiCode int, message string) error {
	switch apiCode {
	case 190:
		return fmt.Errorf("%w: %s (code %d)", domain.ErrAuthentication, message, apiCode)
	case 200, 10:
		return fmt.Errorf("%w: %s (code %d)", domain.ErrAuthorization, message, apiCode)
	default:
		return &domain.APIError{
			Platform: domain.Instagram,
			Code:     apiCode,
			Message:  message,
		}
	}
}

// Platform returns the platform identifier.
func (c *Client) Platform() domain.Platform {
	return domain.Instagram
}
