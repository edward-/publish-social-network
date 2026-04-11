---
name: social-media-publisher
description: Publish media content (images, videos, text) to Facebook, Instagram, TikTok, and YouTube via their official APIs
user-invocable: true
---

# Social Media Publisher CLI — Agent Guide

## Overview

This is a production-ready Go CLI tool for publishing media content (images, videos, text posts) to Facebook, Instagram, TikTok, and YouTube via their official APIs. It follows Clean Architecture principles with clear separation between Domain, UseCase, and Adapter layers.

**Key characteristics:**
- Written in Go 1.21+
- Uses Cobra for CLI framework
- Dependency injection throughout — no global state
- Concurrent publishing to multiple platforms via goroutines
- Domain layer has zero external dependencies

---

## Project Structure

```
social-media-publisher/
├── cmd/publisher/main.go              # CLI entrypoint (Cobra commands)
├── internal/
│   ├── domain/                        # Core business entities
│   │   ├── post.go                   # Post, Platform, MediaType, Publisher interface
│   │   └── errors.go                 # Domain errors (ErrPostValidationFailed, etc.)
│   ├── usecase/                       # Business logic orchestration
│   │   ├── publish.go                # PublishUseCase — concurrent dispatch
│   │   └── publish_test.go           # Unit tests with mocks
│   ├── adapter/                       # Platform-specific implementations
│   │   ├── facebook/                 # Facebook Graph API v18+
│   │   ├── instagram/                # Instagram Graph API
│   │   ├── tiktok/                  # TikTok Content Posting API v2
│   │   └── youtube/                 # YouTube Data API v3
│   └── config/                       # .env loading + validation
├── pkg/media/                         # Shared media file utilities
├── .env.example                       # Credential template
└── README.md                          # User-facing documentation
```

---

## How to Build

```bash
cd social-media-publisher
go build -o publisher ./cmd/publisher
```

This produces a `publisher` binary.

---

## How to Invoke

### Basic Command Structure

```bash
./publisher publish [flags]
```

### Required Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--caption` | `-c` | string | Post caption/description (required) |
| `--platforms` | `-p` | string | Comma-separated: facebook,instagram,tiktok,youtube |

### Optional Flags

| Flag | Type | Description |
|------|------|-------------|
| `--title` | string | Post title (required for YouTube) |
| `--media` | string | Path to local image or video file |
| `--type` | string | Media type: `image`, `video`, `text` (default: `text`) |
| `--tags` | string[] | Tags/hashtags (can repeat: `--tags golang --tags dev`) |
| `--env` | string | Path to .env file (default: `.env`) |
| `--profile` | string | Account profile to use (e.g., 'work', 'personal') |

### Example Invocations

```bash
# Text post to Facebook
./publisher publish --caption "Hello World!" --platforms facebook

# Image to Facebook and Instagram
./publisher publish \
  --caption "Check out this photo!" \
  --media ./photo.jpg \
  --type image \
  --platforms facebook,instagram

# Video to YouTube with title and tags
./publisher publish \
  --title "My Awesome Video" \
  --caption "Video description here" \
  --media ./video.mp4 \
  --type video \
  --platforms youtube \
  --tags golang,programming

# Multi-platform publish
./publisher publish \
  --title "Cross-platform Content" \
  --caption "Published everywhere!" \
  --media ./video.mp4 \
  --type video \
  --platforms youtube,tiktok,facebook \
  --tags golang,devops,programming
```

---

## Understanding the Architecture

### Domain Layer (`internal/domain/`)

This layer has **zero external dependencies**. It defines:

- **Platform** enum: `facebook`, `instagram`, `tiktok`, `youtube`
- **MediaType** enum: `image`, `video`, `text`
- **Post** struct: The core entity with validation logic
- **Publisher** interface: What each adapter must implement
- **Result** struct: Publish outcome (URL or error)

```go
type Publisher interface {
    Publish(ctx context.Context, post Post) (string, error)
    Platform() Platform
    ValidateConfig() error
}
```

### UseCase Layer (`internal/usecase/`)

`PublishUseCase` orchestrates publishing to multiple platforms **concurrently**:

- Takes variadic `...domain.Publisher` in constructor (dependency injection)
- Uses `sync.WaitGroup` + goroutines to publish to all platforms in parallel
- Returns `map[domain.Platform]domain.Result` — partial results allowed (doesn't stop on first error)
- Has its own timeout: 5 minutes total, 2 minutes per platform

### Adapter Layer (`internal/adapter/{platform}/`)

Each platform has a `client.go` (API communication) and `publisher.go` (implements `domain.Publisher`):

| Platform | API Used | Auth Method |
|----------|----------|-------------|
| Facebook | Graph API v18+ | PAGE_ACCESS_TOKEN |
| Instagram | Instagram Graph API | ACCESS_TOKEN (via Facebook) |
| TikTok | Content Posting API v2 | OAuth 2.0 ACCESS_TOKEN |
| YouTube | Data API v3 | OAuth2 (refresh token) |

### Config Layer (`internal/config/`)

Loads `.env` via `godotenv`. Each platform has its own config struct:

```go
type Config struct {
    Facebook  FacebookConfig
    Instagram InstagramConfig
    TikTok    TikTokConfig
    YouTube   YouTubeConfig
}
```

Validation happens at startup — missing required keys for requested platforms cause clear errors.

---

## How to Extend

### Adding a New Platform

1. Create `internal/adapter/newplatform/` directory
2. Implement `client.go` with API communication
3. Implement `publisher.go` implementing `domain.Publisher`:
   ```go
   type Publisher struct {
       client    *Client
       mediaUtil *media.Validator
   }

   func (p *Publisher) Publish(ctx context.Context, post domain.Post) (string, error)
   func (p *Publisher) Platform() domain.Platform { return domain.NewPlatform }
   func (p *Publisher) ValidateConfig() error
   ```
4. Register in `cmd/publisher/main.go` in the `runPublish` switch statement
5. Add config validation in `internal/config/config.go`

### Adding a New CLI Flag

1. Add the flag variable in `cmd/publisher/main.go`:
   ```go
   var flagNewFlag string
   ```
2. Register it with the publish command:
   ```go
   publishCmd.Flags().StringVar(&flagNewFlag, "newflag", "", "Description")
   ```
3. Use it in `runPublish` function

### Modifying Platform Behavior

- **Facebook/TikTok/Instagram**: Edit `internal/adapter/{platform}/publisher.go`
- **YouTube**: Edit `internal/adapter/youtube/client.go` or `publisher.go`

---

## Configuration

Copy `.env.example` to `.env` and fill in credentials:

```bash
cp .env.example .env
```

### Credential Requirements

| Platform | Required Keys |
|----------|--------------|
| Facebook | `FACEBOOK_PAGE_ID`, `FACEBOOK_ACCESS_TOKEN` |
| Instagram | `INSTAGRAM_USER_ID`, `INSTAGRAM_ACCESS_TOKEN` |
| TikTok | `TIKTOK_ACCESS_TOKEN`, `TIKTOK_CLIENT_KEY` |
| YouTube | `YOUTUBE_CLIENT_ID`, `YOUTUBE_CLIENT_SECRET`, `YOUTUBE_REFRESH_TOKEN` |

**Note on TikTok/Instagram auth**: These platforms require browser-based OAuth consent flows. See `README.md` for manual token exchange instructions.

### Multi-Account Support

Configure multiple accounts per platform using profile suffixes:

| Platform | Base Keys | With `--profile=work` |
|----------|-----------|----------------------|
| Facebook | `FACEBOOK_PAGE_ID`, `FACEBOOK_ACCESS_TOKEN` | `FACEBOOK_PAGE_ID_WORK`, `FACEBOOK_ACCESS_TOKEN_WORK` |
| Instagram | `INSTAGRAM_USER_ID`, `INSTAGRAM_ACCESS_TOKEN` | `INSTAGRAM_USER_ID_WORK`, `INSTAGRAM_ACCESS_TOKEN_WORK` |
| TikTok | `TIKTOK_ACCESS_TOKEN`, `TIKTOK_CLIENT_KEY` | `TIKTOK_ACCESS_TOKEN_WORK`, `TIKTOK_CLIENT_KEY_WORK` |
| YouTube | `YOUTUBE_CLIENT_ID`, `YOUTUBE_CLIENT_SECRET`, `YOUTUBE_REFRESH_TOKEN` | `YOUTUBE_CLIENT_ID_WORK`, etc. |

Example `.env`:
```bash
# Default account
FACEBOOK_PAGE_ID=default_id
FACEBOOK_ACCESS_TOKEN=default_token

# Work account
FACEBOOK_PAGE_ID_WORK=work_id
FACEBOOK_ACCESS_TOKEN_WORK=work_token
```

Usage:
```bash
# Use default account
./publisher publish --caption "Hello!" --platforms facebook

# Use work account
./publisher publish --caption "Hello from work!" --platforms facebook --profile work
```

---

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./internal/usecase/... -v

# Run with coverage
go test -cover ./...
```

Unit tests in `internal/usecase/publish_test.go` use mock publishers to test the concurrent dispatch logic.

---

## Error Handling Patterns

- All errors wrapped with `fmt.Errorf("context: %w", err)`
- Domain errors defined in `internal/domain/errors.go`:
  - `ErrPostValidationFailed`
  - `ErrMediaNotFound`
  - `ErrUnsupportedMediaType`
  - `ErrPlatformNotSupported`
  - `ErrAuthentication`
  - `ErrAuthorization`
  - `ErrPublishFailed`
  - `ErrTimeout`
  - `ErrConfigMissing`

---

## Important Implementation Notes

1. **Timeouts**: The use case enforces a 5-minute total timeout and 2-minute per-platform timeout via `context.WithTimeout`.

2. **Partial Results**: Publishing continues to all requested platforms even if some fail. Results are collected via a buffered channel.

3. **Media Validation**: The `pkg/media.Validator` checks file existence, type, and size limits (100MB images, 2GB videos).

4. **Instagram Limitation**: Instagram Graph API requires images/videos to be hosted on a public URL. The current adapter returns an error for local file paths.

5. **YouTube OAuth2**: Uses refresh token flow — no manual token renewal needed after initial setup.

6. **No Global State**: All configuration is injected via constructors. `NewPublisher(cfg)`, `NewPublishUseCase(publishers...)`, etc.
