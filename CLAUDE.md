# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test, Lint

```bash
# Build
go build -o publisher ./cmd/publisher

# Run (no binary)
go run ./cmd/publisher [command]

# All tests
go test ./...

# Tests for a single package (where the unit tests live)
go test ./internal/usecase/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./internal/usecase/...

# Lint / vet
go vet ./...
```

Go 1.21+. Module path: `github.com/edward-/publish-social-network`. Direct deps: `cobra` (CLI), `godotenv` (.env loading), `golang.org/x/oauth2`, `google.golang.org/api` (YouTube).

## CLI Commands

Cobra-based entrypoint at `cmd/publisher/main.go`. Four subcommands:

- `publish` — main command. Required flags: `--caption` (`-c`), `--platforms` (`-p`, comma-separated). Optional: `--title`, `--media` (`-m`), `--type` (`-t`, default `text`), `--tags` (repeatable), `--env` (default `.env`).
- `login-tiktok` — runs TikTok OAuth PKCE flow, saves tokens to `.env`.
- `login-facebook` — runs Facebook OAuth flow, saves access token to `.env`.
- `login-youtube` — runs Google OAuth2 flow, saves refresh token to `.env`.

The `login-*` commands start a local callback HTTP server, open the OS browser to the auth URL, wait for the redirect with the auth code, exchange for tokens, and write them back into `.env`.

## Architecture — Clean Architecture, 3 Layers

```
cmd/publisher/main.go          # CLI wiring only (Cobra commands, flag parsing, result printing)
internal/
  domain/                      # Pure entities + interfaces, ZERO external deps
  usecase/                     # Business logic (concurrent dispatch, validation)
  adapter/{platform}/          # Platform API implementations
internal/config/               # .env loading and validation
pkg/media/                     # Cross-cutting media file utilities (validation, MIME)
```

The dependency direction is strict: `cmd` → `usecase` → `domain` ← `adapter`. Adapters depend on `domain` (to implement `Publisher`); the use case depends only on `domain` interfaces and is unaware of any specific platform.

### Domain (`internal/domain/`)
- `Platform` enum: `facebook`, `instagram`, `tiktok`, `youtube`
- `MediaType` enum: `image`, `video`, `text`
- `Post` struct with `Validate()` (caption required; media path required for image/video; title required for YouTube)
- `Result{Platform, PostID, URL, Error}` with `Success()` helper
- `Publisher` interface — what every adapter implements:
  ```go
  type Publisher interface {
      Publish(ctx context.Context, post Post) (string, error)
      Platform() Platform
      ValidateConfig() error
  }
  ```
- Domain errors: `ErrPostValidationFailed`, `ErrMediaNotFound`, `ErrUnsupportedMediaType`, `ErrPlatformNotSupported`, `ErrAuthentication`, `ErrAuthorization`, `ErrPublishFailed`, `ErrTimeout`, `ErrConfigMissing`. `ValidationError` wraps multiple errors. `APIError` wraps platform HTTP errors.

### UseCase (`internal/usecase/`)
- `PublishUseCase` takes variadic `...Publisher` in constructor (DI). Builds a `map[Platform]Publisher`.
- `Publish(ctx, post)` validates first, then dispatches one goroutine per platform with `sync.WaitGroup`, collects results on a buffered channel, and returns `map[Platform]Result`. **Failures on one platform do not stop others.**
- Timeouts (constants in `publish.go`): 5 min total, 2 min per platform. Applied via `context.WithTimeout` per goroutine.

### Adapters (`internal/adapter/{platform}/`)
Each platform follows the same pattern: `client.go` (HTTP/SDK transport, auth handling) and `publisher.go` (implements `domain.Publisher`, owns a `*Client` and a `*media.Validator`).

| Platform | API | Auth |
|---|---|---|
| Facebook | Graph API | Page access token |
| Instagram | Graph API (via Facebook) | Access token (requires public-URL media — see below) |
| TikTok | Content Posting API v2 | OAuth2 access + refresh token |
| YouTube | Data API v3 | OAuth2 refresh token (auto-renewed via `oauth2` lib) |

Adapter capabilities are not uniform — see "Platform Quirks" below.

### Config (`internal/config/`)
- `Load(envPath)` reads `.env` via `godotenv` and pulls each base key directly with `os.Getenv`.
- `Validate(requestedPlatforms)` checks required keys per platform before publishing starts.
- Per-platform `ValidateXxx()` helpers exist but the main `Validate` is what `runPublish` calls.

### `pkg/media/`
- `Validator` reads files from disk, checks size limits (100MB image, 2GB video), infers MIME from extension. `ReadAndValidate` returns `(MediaInfo, io.ReadCloser, error)`.
- The `IsImage`/`IsVideo` helpers construct a fresh validator each call (cheap, stateless).

## Conventions and Patterns

- **No global state.** Every dependency flows through constructors: `NewPublisher(cfg)`, `NewPublishUseCase(publishers...)`, `NewValidator()`.
- **Error wrapping is consistent**: `fmt.Errorf("context: %w", err)`. Adapters that wrap an API error use the domain `APIError` struct.
- **Partial results are a feature**, not a bug — multi-platform publishes are fire-and-forget per platform. The result map always contains an entry for every requested platform.
- **YouTube uploads are wrapped in a goroutine** with a `select { case <-ctx.Done(): ... case r := <-resultCh: ... }` so context cancellation cleanly aborts the upload. TikTok uses the same pattern.
- **The use case validates the post** before launching any goroutine, so a bad post never partially publishes.
- **`ValidationError` accumulates field errors** rather than returning the first one — `Post.Validate()` collects into a slice.

## Platform Quirks (non-obvious, easy to trip on)

- **Instagram Graph API requires media to be hosted on a public URL.** The current `instagram.Publisher.publishImage` / `publishVideo` return an error for local files — they demonstrate the two-step container-creation flow but don't implement temp hosting. `publishText` always errors (Instagram has no text-only posts). To make Instagram work end-to-end, the gap to fill is temp hosting (S3/GCS) and feeding the resulting URL into the container API.
- **Facebook does not support direct video upload via the Graph API** — `facebook.Publisher.Publish` returns `ErrUnsupportedMediaType` for `MediaTypeVideo`. Text and image only.
- **YouTube only accepts video** — returns `ErrUnsupportedMediaType` for non-video posts. Defaults to `privacyStatus: "private"` in `youtube/publisher.go`; change there if needed.
- **TikTok only accepts video** — same pattern, returns `ErrUnsupportedMediaType` for non-video. `Publisher.RefreshToken()` exists for manual refresh; auto-refresh is not currently wired (`refreshTokenIfNeeded` is a no-op stub).
- **YouTube privacy status is hardcoded** to `"private"`. To make a video public/unlisted, edit `youtube/publisher.go` (the `privacyStatus` variable in `Publish`).

## Adding a New Platform

1. Create `internal/adapter/newplatform/` with `client.go` (transport) and `publisher.go` (implements `domain.Publisher`). Use `media.NewValidator()` for file handling.
2. Add `NewPlatformConfig` struct in `internal/config/config.go`, populate it in `Load`, and add a case in `Validate` for the new platform's required keys.
3. Add a `domain.NewPlatform Platform = "newplatform"` constant and a case in `Platform.IsValid()`.
4. Register the constructor in `cmd/publisher/main.go`'s `runPublish` switch (the one that builds publishers from requested platforms).
5. If the platform needs OAuth login from the CLI, add a new Cobra subcommand modeled on `runTikTokLogin` / `runFacebookLogin` / `runYouTubeLogin` — they all follow the same flow: start local callback server, open browser, wait for code, exchange, write token to `.env`.

## Adding a New CLI Flag

In `cmd/publisher/main.go`: add a package-level `var`, register with `publishCmd.Flags().StringVarP(...)` (or `StringSliceVar` for repeatable), and consume it inside `runPublish`.

## Testing Notes

- `internal/usecase/publish_test.go` uses an in-package `MockPublisher` (so it can read internal fields). Tests cover: registration, success, partial failure, validation error, unsupported platform, concurrent dispatch, and `Post.Validate` table cases.
- No integration tests; adapter correctness is verified by the live API flows behind the `login-*` subcommands.
- The use case's timeouts (5 min / 2 min) are real — when debugging, pass an already-cancelled context to short-circuit.
