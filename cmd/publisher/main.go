// Package main is the entry point for the social media publisher CLI.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

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

	// Create tiktok login command
	tiktokLoginCmd := &cobra.Command{
		Use:   "login-tiktok",
		Short: "Authenticate with TikTok OAuth",
		Long: `Opens a browser for TikTok OAuth authentication and stores the access token.
This command implements TikTok's Login Kit for Desktop using PKCE authorization flow.`,
		RunE: runTikTokLogin,
	}
	tiktokLoginCmd.Flags().StringVar(&flagEnvPath, "env", ".env", "Path to .env file")
	tiktokLoginCmd.Flags().StringVar(&flagProfile, "profile", "", "Account profile to save token (e.g., 'work', 'personal')")
	tiktokLoginCmd.Flags().String("redirect-uri", "http://localhost:8989", "OAuth redirect URI (must match TikTok app config)")

	rootCmd.AddCommand(tiktokLoginCmd)

	// Create facebook login command
	facebookLoginCmd := &cobra.Command{
		Use:   "login-facebook",
		Short: "Authenticate with Facebook OAuth",
		Long: `Opens a browser for Facebook OAuth authentication and stores the access token.
This command implements Facebook's OAuth 2.0 flow for user authentication.`,
		RunE: runFacebookLogin,
	}
	facebookLoginCmd.Flags().StringVar(&flagEnvPath, "env", ".env", "Path to .env file")
	facebookLoginCmd.Flags().StringVar(&flagProfile, "profile", "", "Account profile to save token (e.g., 'work', 'personal')")

	rootCmd.AddCommand(facebookLoginCmd)

	// Create youtube login command
	youtubeLoginCmd := &cobra.Command{
		Use:   "login-youtube",
		Short: "Authenticate with YouTube OAuth",
		Long: `Opens a browser for YouTube OAuth authentication and stores the refresh token.
This command implements Google's OAuth 2.0 flow for YouTube API access.`,
		RunE: runYouTubeLogin,
	}
	youtubeLoginCmd.Flags().StringVar(&flagEnvPath, "env", ".env", "Path to .env file")
	youtubeLoginCmd.Flags().StringVar(&flagProfile, "profile", "", "Account profile to save token (e.g., 'work', 'personal')")

	rootCmd.AddCommand(youtubeLoginCmd)

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

// runTikTokLogin handles the TikTok OAuth login flow.
func runTikTokLogin(cmd *cobra.Command, args []string) error {
	// Load config to get client key
	cfg, err := config.Load(flagEnvPath, flagProfile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.TikTok.ClientKey == "" {
		return fmt.Errorf("TIKTOK_CLIENT_KEY is required in .env file")
	}

	// Generate PKCE pair
	verifier, challenge, err := tiktok.GeneratePKCE()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE: %w", err)
	}
	fmt.Printf("DEBUG: Generated verifier: %s (len=%d)\n", verifier, len(verifier))
	fmt.Printf("DEBUG: Generated challenge: %s\n", challenge)
	fmt.Printf("DEBUG: PKCE verification: %v\n", tiktok.VerifyPKCE(verifier, challenge))

	// Generate CSRF state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Start callback server FIRST so we know the actual port
	// We pass state and verifier so it can validate the callback
	callbackServer, err := tiktok.NewCallbackServer(state, verifier)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer callbackServer.Close()

	fmt.Printf("Listening for callback at: %s\n", callbackServer.URL())

	// Build OAuth config with actual callback URL
	oauthCfg := tiktok.OAuthConfig{
		ClientKey:    cfg.TikTok.ClientKey,
		ClientSecret: cfg.TikTok.ClientSecret,
		RedirectURI:  callbackServer.URL(),
		Scopes:       []string{"user.info.basic", "video.upload", "video.publish"},
	}

	// Build authorization URL
	authURL := tiktok.BuildAuthorizationURL(oauthCfg, state, challenge)
	fmt.Printf("Opening browser for TikTok OAuth...\n")
	fmt.Printf("If browser doesn't open, go to:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Note: Could not open browser automatically: %v\n", err)
		fmt.Println("Please manually open the URL above in your browser.")
	}

	fmt.Println("Waiting for authentication...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	code, err := callbackServer.WaitForCallback(ctx)
	if err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}

	fmt.Println("\nAuthorization code received!")

	// Exchange code for tokens
	tokenResp, err := tiktok.ExchangeCode(oauthCfg, code, verifier)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	tokenInfo := tiktok.ParseTokenResponse(tokenResp)

	fmt.Printf("\n=== TikTok Authentication Successful ===\n\n")
	fmt.Printf("Access Token:  %s...\n", maskString(tokenInfo.AccessToken, 20))
	fmt.Printf("Refresh Token: %s...\n", maskString(tokenInfo.RefreshToken, 20))
	fmt.Printf("Expires At:    %s\n", tokenInfo.ExpiresAt.Format(time.RFC1123))
	fmt.Printf("Open ID:       %s\n", tokenInfo.OpenID)
	fmt.Printf("Scope:         %s\n", tokenInfo.Scope)

	// Save to .env file
	envPath, _ := cmd.Flags().GetString("env")
	if err := saveTikTokToken(envPath, flagProfile, tokenInfo.AccessToken, tokenInfo.RefreshToken); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("\nTokens saved to %s\n", envPath)
	return nil
}

// saveTikTokToken saves the TikTok token to the .env file.
func saveTikTokToken(envPath, profile, accessToken, refreshToken string) error {
	keySuffix := ""
	if profile != "" && profile != "default" {
		keySuffix = "_" + strings.ToUpper(profile)
	}

	// Read existing .env file if it exists
	var lines []string
	if data, err := os.ReadFile(envPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Update or add TIKTOK_ACCESS_TOKEN line
	accessTokenKey := "TIKTOK_ACCESS_TOKEN" + keySuffix
	refreshTokenKey := "TIKTOK_REFRESH_TOKEN" + keySuffix

	foundAccess := false
	foundRefresh := false

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, accessTokenKey+"=") {
			lines[i] = accessTokenKey + "=" + accessToken
			foundAccess = true
		}
		if strings.HasPrefix(line, refreshTokenKey+"=") {
			lines[i] = refreshTokenKey + "=" + refreshToken
			foundRefresh = true
		}
	}

	if !foundAccess {
		lines = append(lines, accessTokenKey+"="+accessToken)
	}
	if !foundRefresh {
		lines = append(lines, refreshTokenKey+"="+refreshToken)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// maskString returns a masked version of a string for display.
func maskString(s string, visible int) string {
	if len(s) <= visible {
		return s
	}
	return s[:visible] + "..."
}

// openBrowser attempts to open the default browser.
func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = runCommand("open", url)
	case "linux":
		err = runCommand("xdg-open", url)
	case "windows":
		err = runCommand("cmd", "/c", "start", url)
	default:
		err = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return err
}

// runCommand runs a command with arguments.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runFacebookLogin handles the Facebook OAuth login flow.
func runFacebookLogin(cmd *cobra.Command, args []string) error {
	// Load config to get client ID and secret
	cfg, err := config.Load(flagEnvPath, flagProfile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Facebook.ClientID == "" {
		return fmt.Errorf("FACEBOOK_CLIENT_ID is required in .env file")
	}
	if cfg.Facebook.ClientSecret == "" {
		return fmt.Errorf("FACEBOOK_CLIENT_SECRET is required in .env file")
	}

	// Generate CSRF state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Start callback server
	callbackServer, err := facebook.NewCallbackServer(state)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer callbackServer.Close()

	fmt.Printf("Listening for callback at: %s\n", callbackServer.URL())

	// Build OAuth config
	oauthCfg := facebook.OAuthConfig{
		ClientID:    cfg.Facebook.ClientID,
		ClientSecret: cfg.Facebook.ClientSecret,
		RedirectURI:  callbackServer.URL(),
		Scopes:       []string{"pages_manage_posts", "pages_read_engagement", "public_profile"},
	}

	// Build authorization URL
	authURL := facebook.BuildAuthorizationURL(oauthCfg, state)
	fmt.Printf("Opening browser for Facebook OAuth...\n")
	fmt.Printf("If browser doesn't open, go to:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Note: Could not open browser automatically: %v\n", err)
		fmt.Println("Please manually open the URL above in your browser.")
	}

	fmt.Println("Waiting for authentication...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	code, err := callbackServer.WaitForCallback(ctx)
	if err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}

	fmt.Println("\nAuthorization code received!")

	// Exchange code for tokens
	tokenResp, err := facebook.ExchangeCode(oauthCfg, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	tokenInfo := facebook.ParseTokenResponse(tokenResp)

	fmt.Printf("\n=== Facebook Authentication Successful ===\n\n")
	fmt.Printf("Access Token: %s...\n", maskString(tokenInfo.AccessToken, 20))
	fmt.Printf("Expires At:   %s\n", tokenInfo.ExpiresAt.Format(time.RFC1123))

	// Save to .env file
	envPath, _ := cmd.Flags().GetString("env")
	if err := saveFacebookToken(envPath, flagProfile, tokenInfo.AccessToken); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("\nToken saved to %s\n", envPath)
	return nil
}

// saveFacebookToken saves the Facebook token to the .env file.
func saveFacebookToken(envPath, profile, accessToken string) error {
	keySuffix := ""
	if profile != "" && profile != "default" {
		keySuffix = "_" + strings.ToUpper(profile)
	}

	// Read existing .env file if it exists
	var lines []string
	if data, err := os.ReadFile(envPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Update or add FACEBOOK_ACCESS_TOKEN line
	tokenKey := "FACEBOOK_ACCESS_TOKEN" + keySuffix

	found := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, tokenKey+"=") {
			lines[i] = tokenKey + "=" + accessToken
			found = true
		}
	}

	if !found {
		lines = append(lines, tokenKey+"="+accessToken)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// runYouTubeLogin handles the YouTube OAuth login flow.
func runYouTubeLogin(cmd *cobra.Command, args []string) error {
	// Load config to get client ID and secret
	cfg, err := config.Load(flagEnvPath, flagProfile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.YouTube.ClientID == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_ID is required in .env file")
	}
	if cfg.YouTube.ClientSecret == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_SECRET is required in .env file")
	}

	// Generate CSRF state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Start callback server
	callbackServer, err := youtube.NewCallbackServer(state)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer callbackServer.Close()

	fmt.Printf("Listening for callback at: %s\n", callbackServer.URL())

	// Build OAuth config
	oauthCfg := youtube.DefaultOAuthConfig(
		cfg.YouTube.ClientID,
		cfg.YouTube.ClientSecret,
		callbackServer.URL(),
	)

	// Build authorization URL
	authURL := youtube.AuthURL(oauthCfg, state)
	fmt.Printf("Opening browser for YouTube OAuth...\n")
	fmt.Printf("If browser doesn't open, go to:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Note: Could not open browser automatically: %v\n", err)
		fmt.Println("Please manually open the URL above in your browser.")
	}

	fmt.Println("Waiting for authentication...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	code, err := callbackServer.WaitForCallback(ctx)
	if err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}

	fmt.Println("\nAuthorization code received!")

	// Exchange code for tokens
	token, err := youtube.ExchangeCode(oauthCfg, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	fmt.Printf("\n=== YouTube Authentication Successful ===\n\n")
	fmt.Printf("Access Token:  %s...\n", maskString(token.AccessToken, 20))
	fmt.Printf("Refresh Token: %s...\n", maskString(token.RefreshToken, 20))
	fmt.Printf("Expires At:    %s\n", token.Expiry.Format(time.RFC1123))

	// Save to .env file
	envPath, _ := cmd.Flags().GetString("env")
	if err := saveYouTubeToken(envPath, flagProfile, token.RefreshToken); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("\nToken saved to %s\n", envPath)
	return nil
}

// saveYouTubeToken saves the YouTube refresh token to the .env file.
func saveYouTubeToken(envPath, profile, refreshToken string) error {
	keySuffix := ""
	if profile != "" && profile != "default" {
		keySuffix = "_" + strings.ToUpper(profile)
	}

	// Read existing .env file if it exists
	var lines []string
	if data, err := os.ReadFile(envPath); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Update or add YOUTUBE_REFRESH_TOKEN line
	tokenKey := "YOUTUBE_REFRESH_TOKEN" + keySuffix

	found := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, tokenKey+"=") {
			lines[i] = tokenKey + "=" + refreshToken
			found = true
		}
	}

	if !found {
		lines = append(lines, tokenKey+"="+refreshToken)
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
