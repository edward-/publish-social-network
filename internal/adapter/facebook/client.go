// Package facebook provides the Facebook Graph API adapter.
package facebook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/edward-/publish-social-network/internal/domain"
)

// Client is a Facebook Graph API client.
type Client struct {
	pageID      string
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

// NewClient creates a new Facebook API client.
func NewClient(pageID, accessToken string) *Client {
	return &Client{
		pageID:      pageID,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://graph.facebook.com/v18.0",
	}
}

// PhotoResponse represents the response from a photo upload.
type PhotoResponse struct {
	ID      string `json:"id"`
	PostID  string `json:"post_id"`
	Success bool   `json:"success"`
}

// FeedResponse represents the response from a feed post.
type FeedResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

// ErrorResponse represents a Facebook API error.
type ErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// PublishPhoto publishes a photo to the Facebook page.
func (c *Client) PublishPhoto(caption string, imageData []byte) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/photos", c.baseURL, c.pageID)

	// Build form data
	values := url.Values{}
	values.Set("caption", caption)
	values.Set("access_token", c.accessToken)

	// Create multipart request
	body := &bytes.Buffer{}
	writer := newMultipartWriter(body)
	writer.WriteField("caption", caption)
	writer.WriteField("access_token", c.accessToken)
	writer.WriteFile("source", "image.jpg", "image/jpeg", imageData)
	writer.Close()

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", c.mapError(resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var photoResp PhotoResponse
	if err := json.Unmarshal(respBody, &photoResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return fmt.Sprintf("https://www.facebook.com/photo?fb_id=%s", photoResp.ID), nil
}

// PublishFeed publishes a text post to the Facebook page.
func (c *Client) PublishFeed(caption string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/feed", c.baseURL, c.pageID)

	values := url.Values{}
	values.Set("message", caption)
	values.Set("access_token", c.accessToken)

	resp, err := c.httpClient.PostForm(endpoint, values)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", c.mapError(resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feedResp FeedResponse
	if err := json.Unmarshal(respBody, &feedResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return fmt.Sprintf("https://www.facebook.com/%s/posts/%s", c.pageID, feedResp.ID), nil
}

// mapError maps Facebook API errors to domain errors.
func (c *Client) mapError(httpCode, apiCode int, message string) error {
	switch apiCode {
	case 190:
		return fmt.Errorf("%w: %s (code %d)", domain.ErrAuthentication, message, apiCode)
	case 200, 10:
		return fmt.Errorf("%w: %s (code %d)", domain.ErrAuthorization, message, apiCode)
	default:
		return &domain.APIError{
			Platform: domain.Facebook,
			Code:     apiCode,
			Message:  message,
		}
	}
}

// Platform returns the platform identifier.
func (c *Client) Platform() domain.Platform {
	return domain.Facebook
}
