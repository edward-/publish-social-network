// Package domain defines the core entities and interfaces for the social media publisher.
package domain

import (
	"context"
	"fmt"
	"time"
)

// Platform is a supported social network.
type Platform string

const (
	Facebook  Platform = "facebook"
	Instagram Platform = "instagram"
	TikTok    Platform = "tiktok"
	YouTube   Platform = "youtube"
)

// String returns the string representation of a Platform.
func (p Platform) String() string {
	return string(p)
}

// IsValid checks if the platform is a supported value.
func (p Platform) IsValid() bool {
	switch p {
	case Facebook, Instagram, TikTok, YouTube:
		return true
	}
	return false
}

// MediaType describes the content being published.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeText  MediaType = "text"
)

// String returns the string representation of a MediaType.
func (m MediaType) String() string {
	return string(m)
}

// IsValid checks if the media type is a supported value.
func (m MediaType) IsValid() bool {
	switch m {
	case MediaTypeImage, MediaTypeVideo, MediaTypeText:
		return true
	}
	return false
}

// Post is the core domain entity representing content to be published.
type Post struct {
	Title       string
	Caption     string
	MediaPath   string
	MediaType   MediaType
	Tags        []string
	Platforms   []Platform
	CreatedAt   time.Time
}

// Validate validates the Post and returns any validation errors.
func (p *Post) Validate() error {
	var errs []error

	if p.Caption == "" {
		errs = append(errs, fmt.Errorf("caption is required"))
	}

	if !p.MediaType.IsValid() {
		errs = append(errs, fmt.Errorf("invalid media type: %s", p.MediaType))
	}

	if len(p.Platforms) == 0 {
		errs = append(errs, fmt.Errorf("at least one platform is required"))
	}

	for _, platform := range p.Platforms {
		if !platform.IsValid() {
			errs = append(errs, fmt.Errorf("invalid platform: %s", platform))
		}
	}

	// MediaPath is required for image and video posts
	if p.MediaType != MediaTypeText && p.MediaPath == "" {
		errs = append(errs, fmt.Errorf("media path is required for %s posts", p.MediaType))
	}

	// Title is required for YouTube
	for _, platform := range p.Platforms {
		if platform == YouTube && p.Title == "" {
			errs = append(errs, fmt.Errorf("title is required for YouTube posts"))
			break
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	return nil
}

// Result represents the outcome of a publish operation on a single platform.
type Result struct {
	Platform Platform
	PostID   string
	URL      string
	Error    error
}

// Success returns true if the publish operation was successful.
func (r Result) Success() bool {
	return r.Error == nil
}

// Publisher is the interface each platform adapter must implement.
type Publisher interface {
	// Publish publishes a post to the platform and returns the post URL/ID.
	Publish(ctx context.Context, post Post) (string, error)

	// Platform returns the platform this publisher handles.
	Platform() Platform

	// ValidateConfig validates that the publisher has valid configuration.
	ValidateConfig() error
}

// ConfigValidator is an optional interface for publishers to validate config.
type ConfigValidator interface {
	ValidateConfig() error
}
