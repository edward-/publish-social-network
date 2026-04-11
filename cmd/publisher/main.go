// Package main is the entry point for the social media publisher CLI.
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/edward-/publish-social-network/internal/adapter/facebook"
	"github.com/edward-/publish-social-network/internal/adapter/instagram"
	"github.com/edward-/publish-social-network/internal/adapter/tiktok"
	"github.com/edward-/publish-social-network/internal/adapter/youtube"
	"github.com/edward-/publish-social-network/internal/config"
	"github.com/edward-/publish-social-network/internal/domain"
	"github.com/edward-/publish-social-network/internal/usecase"
	"github.com/spf13/cobra"
)

var (
	// CLI flags
	flagTitle     string
	flagCaption   string
	flagMedia     string
	flagType      string
	flagPlatforms string
	flagTags      []string
	flagEnvPath   string
	flagProfile   string
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "publisher",
		Short: "Social Media Publisher - Publish content to multiple platforms",
		Long: `Social Media Publisher is a CLI tool for publishing content
(images, videos, text posts) to Facebook, Instagram, TikTok, and YouTube.`,
	}

	// Create publish command
	publishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish a post to social media platforms",
		RunE:  runPublish,
	}

	// Add flags to publish command
	publishCmd.Flags().StringVar(&flagTitle, "title", "", "Post title (required for YouTube)")
	publishCmd.Flags().StringVarP(&flagCaption, "caption", "c", "", "Post caption/description (required)")
	publishCmd.Flags().StringVarP(&flagMedia, "media", "m", "", "Path to local image or video file")
	publishCmd.Flags().StringVarP(&flagType, "type", "t", "text", "Media type: image | video | text")
	publishCmd.Flags().StringVarP(&flagPlatforms, "platforms", "p", "", "Comma-separated list of platforms: facebook,instagram,tiktok,youtube")
	publishCmd.Flags().StringSliceVar(&flagTags, "tags", []string{}, "Tags/hashtags (can be specified multiple times)")
	publishCmd.Flags().StringVar(&flagEnvPath, "env", ".env", "Path to .env file")
	publishCmd.Flags().StringVar(&flagProfile, "profile", "", "Account profile to use (e.g., 'work', 'personal')")

	// Mark required flags
	publishCmd.MarkFlagRequired("caption")
	publishCmd.MarkFlagRequired("platforms")

	rootCmd.AddCommand(publishCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runPublish is the main logic for the publish command.
func runPublish(cmd *cobra.Command, args []string) error {
	// Parse platforms
	platforms := parsePlatforms(flagPlatforms)
	if len(platforms) == 0 {
		return fmt.Errorf("no valid platforms specified")
	}

	// Parse media type
	mediaType := parseMediaType(flagType)

	// Load and validate configuration
	cfg, err := config.Load(flagEnvPath, flagProfile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert platforms to strings for validation
	platformStrs := make([]string, len(platforms))
	for i, p := range platforms {
		platformStrs[i] = string(p)
	}

	// Validate config for requested platforms
	if err := cfg.Validate(platformStrs); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Create post
	post := domain.Post{
		Title:     flagTitle,
		Caption:   flagCaption,
		MediaPath: flagMedia,
		MediaType: mediaType,
		Tags:      flagTags,
		Platforms: platforms,
	}

	// Create publishers based on requested platforms
	var publishers []domain.Publisher

	for _, platform := range platforms {
		var publisher domain.Publisher
		var err error

		switch platform {
		case domain.Facebook:
			publisher = facebook.NewPublisher(cfg.Facebook)
		case domain.Instagram:
			publisher = instagram.NewPublisher(cfg.Instagram)
		case domain.TikTok:
			publisher = tiktok.NewPublisher(cfg.TikTok)
		case domain.YouTube:
			publisher, err = youtube.NewPublisher(cfg.YouTube)
			if err != nil {
				return fmt.Errorf("failed to create YouTube publisher: %w", err)
			}
		}

		if publisher != nil {
			if err := publisher.ValidateConfig(); err != nil {
				return fmt.Errorf("invalid %s config: %w", platform, err)
			}
			publishers = append(publishers, publisher)
		}
	}

	// Create use case and publish
	uc := usecase.NewPublishUseCase(publishers...)
	results := uc.Publish(cmd.Context(), post)

	// Print results as table
	printResults(results)

	return nil
}

// parsePlatforms parses a comma-separated list of platform names.
func parsePlatforms(platformsStr string) []domain.Platform {
	var platforms []domain.Platform
	parts := strings.Split(platformsStr, ",")

	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		switch domain.Platform(p) {
		case domain.Facebook:
			platforms = append(platforms, domain.Facebook)
		case domain.Instagram:
			platforms = append(platforms, domain.Instagram)
		case domain.TikTok:
			platforms = append(platforms, domain.TikTok)
		case domain.YouTube:
			platforms = append(platforms, domain.YouTube)
		}
	}

	return platforms
}

// parseMediaType parses a media type string.
func parseMediaType(mediaTypeStr string) domain.MediaType {
	switch strings.ToLower(mediaTypeStr) {
	case "image":
		return domain.MediaTypeImage
	case "video":
		return domain.MediaTypeVideo
	default:
		return domain.MediaTypeText
	}
}

// printResults prints the publish results as a formatted table.
func printResults(results map[domain.Platform]domain.Result) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(writer, "Platform\tStatus\tURL / Error")
	fmt.Fprintln(writer, "--------\t------\t------------------")

	// Print each result
	for platform, result := range results {
		status := "OK"
		urlOrError := result.URL

		if result.Error != nil {
			status = "FAILED"
			urlOrError = result.Error.Error()
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\n", platform, status, urlOrError)
	}

	writer.Flush()
}
