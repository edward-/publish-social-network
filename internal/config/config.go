// Package config handles loading and validating configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all platform-specific configuration.
type Config struct {
	Facebook  FacebookConfig
	Instagram InstagramConfig
	TikTok    TikTokConfig
	YouTube   YouTubeConfig
}

// FacebookConfig holds Facebook API configuration.
type FacebookConfig struct {
	PageID       string
	AccessToken  string
	ClientID     string
	ClientSecret string
}

// InstagramConfig holds Instagram API configuration.
type InstagramConfig struct {
	UserID      string
	AccessToken string
}

// TikTokConfig holds TikTok API configuration.
type TikTokConfig struct {
	AccessToken  string
	RefreshToken string
	ClientKey    string
	ClientSecret string
}

// YouTubeConfig holds YouTube API configuration.
type YouTubeConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// Load reads configuration from the .env file and environment variables.
func Load(envPath string) (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load(envPath)

	return &Config{
		Facebook: FacebookConfig{
			PageID:       os.Getenv("FACEBOOK_PAGE_ID"),
			AccessToken:  os.Getenv("FACEBOOK_ACCESS_TOKEN"),
			ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
			ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
		},
		Instagram: InstagramConfig{
			UserID:      os.Getenv("INSTAGRAM_USER_ID"),
			AccessToken: os.Getenv("INSTAGRAM_ACCESS_TOKEN"),
		},
		TikTok: TikTokConfig{
			AccessToken:  os.Getenv("TIKTOK_ACCESS_TOKEN"),
			RefreshToken: os.Getenv("TIKTOK_REFRESH_TOKEN"),
			ClientKey:    os.Getenv("TIKTOK_CLIENT_KEY"),
			ClientSecret: os.Getenv("TIKTOK_CLIENT_SECRET"),
		},
		YouTube: YouTubeConfig{
			ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
			ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
			RefreshToken: os.Getenv("YOUTUBE_REFRESH_TOKEN"),
		},
	}, nil
}

// Validate checks that all required fields are present for the requested platforms.
func (c *Config) Validate(requestedPlatforms []string) error {
	var missing []string

	for _, platform := range requestedPlatforms {
		switch strings.ToLower(platform) {
		case "facebook":
			if c.Facebook.PageID == "" {
				missing = append(missing, "FACEBOOK_PAGE_ID")
			}
			if c.Facebook.AccessToken == "" {
				missing = append(missing, "FACEBOOK_ACCESS_TOKEN")
			}
		case "instagram":
			if c.Instagram.UserID == "" {
				missing = append(missing, "INSTAGRAM_USER_ID")
			}
			if c.Instagram.AccessToken == "" {
				missing = append(missing, "INSTAGRAM_ACCESS_TOKEN")
			}
		case "tiktok":
			if c.TikTok.AccessToken == "" {
				missing = append(missing, "TIKTOK_ACCESS_TOKEN")
			}
		case "youtube":
			if c.YouTube.ClientID == "" {
				missing = append(missing, "YOUTUBE_CLIENT_ID")
			}
			if c.YouTube.ClientSecret == "" {
				missing = append(missing, "YOUTUBE_CLIENT_SECRET")
			}
			if c.YouTube.RefreshToken == "" {
				missing = append(missing, "YOUTUBE_REFRESH_TOKEN")
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateFacebook returns an error if Facebook config is invalid.
func (c *Config) ValidateFacebook() error {
	if c.Facebook.PageID == "" {
		return fmt.Errorf("FACEBOOK_PAGE_ID is required")
	}
	if c.Facebook.AccessToken == "" {
		return fmt.Errorf("FACEBOOK_ACCESS_TOKEN is required")
	}
	return nil
}

// ValidateInstagram returns an error if Instagram config is invalid.
func (c *Config) ValidateInstagram() error {
	if c.Instagram.UserID == "" {
		return fmt.Errorf("INSTAGRAM_USER_ID is required")
	}
	if c.Instagram.AccessToken == "" {
		return fmt.Errorf("INSTAGRAM_ACCESS_TOKEN is required")
	}
	return nil
}

// ValidateTikTok returns an error if TikTok config is invalid.
func (c *Config) ValidateTikTok() error {
	if c.TikTok.AccessToken == "" {
		return fmt.Errorf("TIKTOK_ACCESS_TOKEN is required")
	}
	return nil
}

// ValidateYouTube returns an error if YouTube config is invalid.
func (c *Config) ValidateYouTube() error {
	if c.YouTube.ClientID == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_ID is required")
	}
	if c.YouTube.ClientSecret == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_SECRET is required")
	}
	if c.YouTube.RefreshToken == "" {
		return fmt.Errorf("YOUTUBE_REFRESH_TOKEN is required")
	}
	return nil
}
