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
	Profile   string // The profile used to load this config
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
// The profile parameter selects which account to use (e.g., "work", "personal").
// An empty profile or "default" uses the base keys without suffix.
func Load(envPath, profile string) (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load(envPath)

	cfg := &Config{
		Profile: profile,
		Facebook: FacebookConfig{
			PageID:       getEnv("FACEBOOK_PAGE_ID", profile),
			AccessToken:  getEnv("FACEBOOK_ACCESS_TOKEN", profile),
			ClientID:     getEnv("FACEBOOK_CLIENT_ID", profile),
			ClientSecret: getEnv("FACEBOOK_CLIENT_SECRET", profile),
		},
		Instagram: InstagramConfig{
			UserID:      getEnv("INSTAGRAM_USER_ID", profile),
			AccessToken: getEnv("INSTAGRAM_ACCESS_TOKEN", profile),
		},
		TikTok: TikTokConfig{
			AccessToken:  getEnv("TIKTOK_ACCESS_TOKEN", profile),
			RefreshToken: getEnv("TIKTOK_REFRESH_TOKEN", profile),
			ClientKey:    getEnv("TIKTOK_CLIENT_KEY", profile),
			ClientSecret: getEnv("TIKTOK_CLIENT_SECRET", profile),
		},
		YouTube: YouTubeConfig{
			ClientID:     getEnv("YOUTUBE_CLIENT_ID", profile),
			ClientSecret: getEnv("YOUTUBE_CLIENT_SECRET", profile),
			RefreshToken: getEnv("YOUTUBE_REFRESH_TOKEN", profile),
		},
	}

	return cfg, nil
}

// Validate checks that all required fields are present for the requested platforms.
func (c *Config) Validate(requestedPlatforms []string) error {
	var missing []string

	for _, platform := range requestedPlatforms {
		switch strings.ToLower(platform) {
		case "facebook":
			if c.Facebook.PageID == "" {
				missing = append(missing, envKey("FACEBOOK_PAGE_ID", c.Profile))
			}
			if c.Facebook.AccessToken == "" {
				missing = append(missing, envKey("FACEBOOK_ACCESS_TOKEN", c.Profile))
			}
		case "instagram":
			if c.Instagram.UserID == "" {
				missing = append(missing, envKey("INSTAGRAM_USER_ID", c.Profile))
			}
			if c.Instagram.AccessToken == "" {
				missing = append(missing, envKey("INSTAGRAM_ACCESS_TOKEN", c.Profile))
			}
		case "tiktok":
			if c.TikTok.AccessToken == "" {
				missing = append(missing, envKey("TIKTOK_ACCESS_TOKEN", c.Profile))
			}
		case "youtube":
			if c.YouTube.ClientID == "" {
				missing = append(missing, envKey("YOUTUBE_CLIENT_ID", c.Profile))
			}
			if c.YouTube.ClientSecret == "" {
				missing = append(missing, envKey("YOUTUBE_CLIENT_SECRET", c.Profile))
			}
			if c.YouTube.RefreshToken == "" {
				missing = append(missing, envKey("YOUTUBE_REFRESH_TOKEN", c.Profile))
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
		return fmt.Errorf("%s is required", envKey("FACEBOOK_PAGE_ID", c.Profile))
	}
	if c.Facebook.AccessToken == "" {
		return fmt.Errorf("%s is required", envKey("FACEBOOK_ACCESS_TOKEN", c.Profile))
	}
	return nil
}

// ValidateInstagram returns an error if Instagram config is invalid.
func (c *Config) ValidateInstagram() error {
	if c.Instagram.UserID == "" {
		return fmt.Errorf("%s is required", envKey("INSTAGRAM_USER_ID", c.Profile))
	}
	if c.Instagram.AccessToken == "" {
		return fmt.Errorf("%s is required", envKey("INSTAGRAM_ACCESS_TOKEN", c.Profile))
	}
	return nil
}

// ValidateTikTok returns an error if TikTok config is invalid.
func (c *Config) ValidateTikTok() error {
	if c.TikTok.AccessToken == "" {
		return fmt.Errorf("%s is required", envKey("TIKTOK_ACCESS_TOKEN", c.Profile))
	}
	return nil
}

// ValidateYouTube returns an error if YouTube config is invalid.
func (c *Config) ValidateYouTube() error {
	if c.YouTube.ClientID == "" {
		return fmt.Errorf("%s is required", envKey("YOUTUBE_CLIENT_ID", c.Profile))
	}
	if c.YouTube.ClientSecret == "" {
		return fmt.Errorf("%s is required", envKey("YOUTUBE_CLIENT_SECRET", c.Profile))
	}
	if c.YouTube.RefreshToken == "" {
		return fmt.Errorf("%s is required", envKey("YOUTUBE_REFRESH_TOKEN", c.Profile))
	}
	return nil
}

// getEnv reads an environment variable, returning an empty string if not set.
// If profile is non-empty and not "default", it first tries the suffixed version
// (e.g., "FACEBOOK_PAGE_ID_WORK") before falling back to the base key.
func getEnv(key, profile string) string {
	if profile != "" && profile != "default" {
		suffix := "_" + strings.ToUpper(profile)
		if val := os.Getenv(key + suffix); val != "" {
			return val
		}
	}
	return os.Getenv(key)
}

// envKey returns the environment variable key with profile suffix if applicable.
func envKey(key, profile string) string {
	if profile != "" && profile != "default" {
		return key + "_" + strings.ToUpper(profile)
	}
	return key
}
