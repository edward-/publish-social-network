# Social Media Publisher CLI

A production-ready Go CLI tool for publishing media content (images, videos, text posts) to Facebook, Instagram, TikTok, and YouTube via their official APIs.

## Features

- **Multi-platform publishing**: Publish to Facebook, Instagram, TikTok, and YouTube simultaneously
- **Concurrent uploads**: Uses goroutines for parallel publishing across platforms
- **Clean Architecture**: Domain-driven design with clear separation of concerns
- **Dependency Injection**: No global state, all dependencies injected via constructors
- **Type-safe configuration**: Environment-based config with validation

## Prerequisites

### Go Version
- Go 1.21 or later

### API Account Setup

#### Facebook
1. Create a Facebook Developer account at https://developers.facebook.com
2. Create a Facebook App (select "Business" or "Basic" type)
3. Add "Facebook Login" product and configure permissions:
   - `pages_manage_posts`
   - `pages_read_engagement`
4. Generate a Page Access Token via Graph API Explorer:
   - Go to https://developers.facebook.com/tools/explorer/
   - Select your Facebook Page
   - Request the required permissions
   - Generate and copy the token
5. Note: Facebook tokens expire and need periodic renewal

#### Instagram
1. Your Instagram account must be a Business or Creator account
2. Connect it to your Facebook Page (Instagram settings > Account > Linked accounts)
3. Use the Facebook Graph API Explorer to get an Instagram token:
   - Go to https://developers.facebook.com/tools/explorer/
   - Select your app
   - Add permissions: `instagram_basic`, `instagram_content_publish`, `pages_read_engagement`
   - Get your IG User ID from: `GET /{page-id}?fields=instagram_business_account`

#### TikTok
1. Create a TikTok Developer account at https://developers.tiktok.com
2. Create an app and configure the "Content Posting" capability
3. Request the following scopes:
   - `video.upload`
   - `video.publish`
4. **Manual OAuth Flow Required**:
   - TikTok requires browser-based user consent, so you must:
     a. Build the OAuth authorization URL manually
     b. Open it in a browser
     c. Get the authorization code
     d. Exchange it for an access token via API
   - Example authorization URL:
     ```
     https://www.tiktok.com/v2/auth/authorize/
       ?client_key=YOUR_CLIENT_KEY
       &scope=video.upload,video.publish
       &response_type=code
       &redirect_uri=YOUR_REDIRECT_URI
     ```

#### YouTube
1. Create a Google Cloud project at https://console.cloud.google.com
2. Enable the YouTube Data API v3
3. Create OAuth 2.0 credentials:
   - Application type: "Desktop app" or "Web application"
   - Authorized redirect URIs: `http://localhost`
4. **One-time OAuth Flow**:
   - Use a tool like `oauth2l` or implement a simple auth flow:
   ```bash
   # Example using oauth2l (https://github.com/google/oauth2l)
   oauth2l token --credentials path/to/credentials.json \
     --scope https://www.googleapis.com/auth/youtube.upload
   ```
   - Copy the refresh token to your `.env` file

## Configuration

### 1. Copy the environment template
```bash
cp .env.example .env
```

### 2. Fill in your credentials
Edit `.env` with your API credentials:

```bash
# Facebook
FACEBOOK_PAGE_ID=123456789
FACEBOOK_ACCESS_TOKEN=EAAxxxxxx

# Instagram
INSTAGRAM_USER_ID=17841401234567890
INSTAGRAM_ACCESS_TOKEN=EAAxxxxxx

# TikTok
TIKTOK_ACCESS_TOKEN=act.xxxxxx
TIKTOK_CLIENT_KEY=your_client_key

# YouTube
YOUTUBE_CLIENT_ID=xxx.apps.googleusercontent.com
YOUTUBE_CLIENT_SECRET=GOCSPX-xxx
YOUTUBE_REFRESH_TOKEN=1//xxx
```

### 3. Multi-Account Support (Optional)
If you have multiple accounts per platform, configure them with profile suffixes:

```bash
# Default account
FACEBOOK_PAGE_ID=default_id
FACEBOOK_ACCESS_TOKEN=default_token

# Work account
FACEBOOK_PAGE_ID_WORK=work_id
FACEBOOK_ACCESS_TOKEN_WORK=work_token

# Personal account
FACEBOOK_PAGE_ID_PERSONAL=personal_id
FACEBOOK_ACCESS_TOKEN_PERSONAL=personal_token
```

Then use the `--profile` flag to select which account to use:

```bash
# Use default account
./publisher publish --caption "Hello!" --platforms facebook

# Use work account
./publisher publish --caption "Hello from work!" --platforms facebook --profile work

# Use personal account
./publisher publish --caption "Hello from personal!" --platforms facebook --profile personal
```

## Building

### Build the CLI
```bash
go build -o publisher ./cmd/publisher
```

### Run directly
```bash
go run ./cmd/publisher [command]
```

## Usage

### Basic Commands

```bash
# Publish a text post to Facebook
./publisher publish \
  --caption "Hello from the Social Media Publisher!" \
  --platforms facebook

# Publish an image to Facebook and Instagram
./publisher publish \
  --caption "Check out this photo!" \
  --media ./photo.jpg \
  --type image \
  --platforms facebook,instagram

# Publish a video to YouTube
./publisher publish \
  --title "My Awesome Video" \
  --caption "Description of the video" \
  --media ./video.mp4 \
  --type video \
  --platforms youtube

# Publish to multiple platforms with tags
./publisher publish \
  --title "Amazing Content" \
  --caption "Don't miss this!" \
  --media ./video.mp4 \
  --type video \
  --platforms youtube,tiktok \
  --tags golang,programming,dev
```

### Flag Reference

| Flag | Short | Description | Required |
|------|-------|-------------|----------|
| `--caption` | `-c` | Post caption/description | Yes |
| `--title` | - | Post title (required for YouTube) | For YouTube |
| `--media` | `-m` | Path to image or video file | For media posts |
| `--type` | `-t` | Media type: `image`, `video`, `text` (default: `text`) | No |
| `--platforms` | `-p` | Comma-separated: `facebook`, `instagram`, `tiktok`, `youtube` | Yes |
| `--tags` | - | Tags/hashtags (can be repeated) | No |
| `--env` | - | Path to .env file (default: `.env`) | No |
| `--profile` | - | Account profile to use (e.g., 'work', 'personal') | No |

### Output Format

Results are displayed as a table:

```
Platform    Status    URL / Error
--------    ------    ------------------
youtube     OK        https://youtu.be/abc123
tiktok      FAILED    401 Unauthorized: token expired
```

## Architecture

```
social-media-publisher/
├── cmd/publisher/main.go          # CLI entrypoint (Cobra)
├── internal/
│   ├── domain/                     # Core entities and interfaces
│   │   ├── post.go                # Post, Platform, MediaType, Publisher
│   │   └── errors.go              # Domain errors
│   ├── usecase/                   # Business logic
│   │   ├── publish.go             # PublishUseCase implementation
│   │   └── publish_test.go        # Unit tests
│   ├── adapter/                   # Platform-specific implementations
│   │   ├── facebook/
│   │   ├── instagram/
│   │   ├── tiktok/
│   │   └── youtube/
│   └── config/                    # Configuration loading
└── pkg/media/                     # Shared media utilities
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/usecase/...
```

## Platform-Specific Notes

### Facebook
- Supports: text posts, image posts
- Videos must be uploaded via Facebook's native interface or other methods
- Tokens expire and need periodic renewal

### Instagram
- Supports: single image posts, Reels (video)
- **Important**: Instagram Graph API requires images/videos to be hosted on a public URL
- The current implementation demonstrates the API flow; production use would require a temp hosting step
- Two-step process: create media container → publish

### TikTok
- Supports: video upload only
- **Important**: TikTok requires browser-based OAuth consent flow
- Tokens must have `video.upload` and `video.publish` scopes
- Chunked upload support for large videos

### YouTube
- Supports: video upload with title, description, tags, privacy status
- Default privacy: private (change in `publisher.go` if needed)
- Uses OAuth2 with refresh token (no manual token renewal needed)
- Progress shown during upload

## Error Handling

- Configuration errors are reported before any publishing starts
- Individual platform errors don't stop publishing to other platforms
- All errors are wrapped with context using `fmt.Errorf("...: %w", err)`

## License

MIT
