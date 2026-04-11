// Package tiktok provides the TikTok Content Posting API adapter.
package tiktok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/edward-/publish-social-network/internal/domain"
)

// Client is a TikTok Content Posting API client.
type Client struct {
	accessToken string
	clientKey   string
	httpClient  *http.Client
	baseURL     string
}

// NewClient creates a new TikTok API client.
func NewClient(accessToken, clientKey string) *Client {
	return &Client{
		accessToken: accessToken,
		clientKey:   clientKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		baseURL: "https://open.tiktokapis.com/v2",
	}
}

// InitVideoResponse represents the response from initializing a video upload.
type InitVideoResponse struct {
	Data struct {
		UploadURL string `json:"upload_url"`
		UploadID  string `json:"upload_id"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// PublishResponse represents the response from publishing a video.
type PublishResponse struct {
	Data struct {
		VideoID string `json:"video_id"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ErrorResponse represents a TikTok API error.
type ErrorResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		LogID     string `json:"log_id"`
	} `json:"error"`
}

// InitVideoUpload initializes a video upload and returns the upload URL and ID.
// This is step 1 of the TikTok video publishing process.
func (c *Client) InitVideoUpload(title, description string, tags []string, postSetting string) (*InitVideoResponse, error) {
	endpoint := fmt.Sprintf("%s/post/publish/video/init/", c.baseURL)

	// Build tags array
	hashtags := make([]map[string]string, len(tags))
	for i, tag := range tags {
		hashtags[i] = map[string]string{"name": tag}
	}

	payload := map[string]interface{}{
		"access_token": c.accessToken,
		"app_key":      c.clientKey,
		"title":        title,
		"description":  description,
		"hashtags":     hashtags,
		"post_setting": postSetting, // "public" | "private"
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, c.mapError(errResp.Error.Code, errResp.Error.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var initResp InitVideoResponse
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &initResp, nil
}

// UploadVideoChunk uploads a chunk of video data using URL upload.
func (c *Client) UploadVideoChunk(uploadURL string, data []byte, offset, length int) error {
	req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers for chunked upload
	req.Header.Set("Content-Type", "video/mp4")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("Upload-Offset", fmt.Sprintf("%d", offset))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload chunk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("chunk upload failed with status: %d", resp.StatusCode)
	}

	return nil
}

// PublishVideo publishes the uploaded video.
func (c *Client) PublishVideo(uploadID string) (*PublishResponse, error) {
	endpoint := fmt.Sprintf("%s/post/publish/video/republish/", c.baseURL)

	payload := map[string]interface{}{
		"access_token": c.accessToken,
		"app_key":      c.clientKey,
		"upload_id":    uploadID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, c.mapError(errResp.Error.Code, errResp.Error.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var publishResp PublishResponse
	if err := json.Unmarshal(respBody, &publishResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &publishResp, nil
}

// UploadVideo performs the complete video upload flow.
func (c *Client) UploadVideo(videoData []byte, title, description string, tags []string, privacyLevel string) (string, error) {
	// Step 1: Initialize upload
	initResp, err := c.InitVideoUpload(title, description, tags, privacyLevel)
	if err != nil {
		return "", fmt.Errorf("failed to initialize upload: %w", err)
	}

	// Step 2: Upload video chunks (chunked upload)
	chunkSize := 5 * 1024 * 1024 // 5MB chunks
	for offset := 0; offset < len(videoData); offset += chunkSize {
		end := offset + chunkSize
		if end > len(videoData) {
			end = len(videoData)
		}
		if err := c.UploadVideoChunk(initResp.Data.UploadURL, videoData[offset:end], offset, end-offset); err != nil {
			return "", fmt.Errorf("failed to upload chunk at offset %d: %w", offset, err)
		}
	}

	// Step 3: Publish
	publishResp, err := c.PublishVideo(initResp.Data.UploadID)
	if err != nil {
		return "", fmt.Errorf("failed to publish video: %w", err)
	}

	return fmt.Sprintf("https://www.tiktok.com/@user/video/%s", publishResp.Data.VideoID), nil
}

// mapError maps TikTok API errors to domain errors.
func (c *Client) mapError(code, message string) error {
	switch code {
	case "10007", "10008", "10009":
		return fmt.Errorf("%w: %s (code %s)", domain.ErrAuthentication, message, code)
	case "10202", "10203":
		return fmt.Errorf("%w: %s (code %s)", domain.ErrAuthorization, message, code)
	default:
		return &domain.APIError{
			Platform: domain.TikTok,
			Code:     0,
			Message:  fmt.Sprintf("%s (code %s)", message, code),
		}
	}
}

// Platform returns the platform identifier.
func (c *Client) Platform() domain.Platform {
	return domain.TikTok
}

// GetVideoURL returns the URL for a TikTok video.
func GetVideoURL(videoID string) string {
	return fmt.Sprintf("https://www.tiktok.com/@user/video/%s", videoID)
}
