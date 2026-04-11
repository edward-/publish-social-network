// Package youtube provides the YouTube Data API v3 adapter.
package youtube

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/edward-/publish-social-network/internal/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
)

// Client is a YouTube Data API v3 client.
type Client struct {
	service       *youtube.Service
	refreshToken  string
	clientID      string
	clientSecret  string
	httpClient    *http.Client
}

// NewClient creates a new YouTube API client.
func NewClient(clientID, clientSecret, refreshToken string) (*Client, error) {
	// Create OAuth2 config
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{youtube.YoutubeUploadScope, youtube.YoutubeReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	// Create token source from refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-1 * time.Hour), // Force refresh
	}

	tokenSource := config.TokenSource(context.Background(), token)

	// Create HTTP client with token source
	httpClient := oauth2.NewClient(context.Background(), tokenSource)

	// Create YouTube service
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

// UploadVideo uploads a video to YouTube and returns the video URL.
func (c *Client) UploadVideo(title, description string, tags []string, privacyStatus string, videoData io.Reader, size int64) (string, error) {
	// Create video metadata
	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: description,
			Tags:        tags,
			CategoryId:  "22", // People & Blogs
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: privacyStatus,
		},
	}

	// Create upload call
	call := c.service.Videos.Insert([]string{"snippet", "status"}, video)
	call = call.ProgressUpdater(func(current, total int64) {
		if total > 0 {
			log.Printf("YouTube upload progress: %d / %d bytes (%.1f%%)", current, total, float64(current)/float64(total)*100)
		}
	})

	// Execute upload - the Media function handles content length internally
	// when given a reader that can be seeked
	uploadResp, err := call.Media(videoData, googleapi.ChunkSize(googleapi.DefaultUploadChunkSize)).Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload video: %w", err)
	}

	log.Printf("Successfully uploaded video: %s", uploadResp.Id)

	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", uploadResp.Id), nil
}

// GetVideo returns details about a video.
func (c *Client) GetVideo(videoID string) (*youtube.Video, error) {
	call := c.service.Videos.List([]string{"snippet", "status"}).Id(videoID)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}
	return resp.Items[0], nil
}

// Platform returns the platform identifier.
func (c *Client) Platform() domain.Platform {
	return domain.YouTube
}
