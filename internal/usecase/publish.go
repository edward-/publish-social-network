// Package usecase implements the business logic for publishing posts.
package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/edward-/publish-social-network/internal/domain"
)

// PublishUseCase handles the business logic for publishing content to platforms.
type PublishUseCase struct {
	publishers map[domain.Platform]domain.Publisher
}

// NewPublishUseCase creates a new PublishUseCase with the given publishers.
// All dependencies are injected via the constructor - no global state.
func NewPublishUseCase(publishers ...domain.Publisher) *PublishUseCase {
	publisherMap := make(map[domain.Platform]domain.Publisher)
	for _, p := range publishers {
		publisherMap[p.Platform()] = p
	}
	return &PublishUseCase{
		publishers: publisherMap,
	}
}

// Publish dispatches a Post to all requested platforms concurrently
// and returns a map[Platform]Result containing success URL or error per platform.
// It does not stop on the first error - all platforms are attempted.
func (uc *PublishUseCase) Publish(ctx context.Context, post domain.Post) map[domain.Platform]domain.Result {
	results := make(map[domain.Platform]domain.Result)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Validate the post first
	if err := post.Validate(); err != nil {
		for _, platform := range post.Platforms {
			results[platform] = domain.Result{
				Platform: platform,
				Error:    fmt.Errorf("post validation failed: %w", err),
			}
		}
		return results
	}

	// Set default timeout if not provided
	ctx, cancel := context.WithTimeout(ctx, defaultPublishTimeout)
	defer cancel()

	// Channel to collect results
	resultCh := make(chan domain.Result, len(post.Platforms))

	// Launch goroutines for each platform
	for _, platform := range post.Platforms {
		publisher, ok := uc.publishers[platform]
		if !ok {
			results[platform] = domain.Result{
				Platform: platform,
				Error:    fmt.Errorf("%w: %s", domain.ErrPlatformNotSupported, platform),
			}
			continue
		}

		wg.Add(1)
		go func(p domain.Publisher, pst domain.Post) {
			defer wg.Done()

			// Create a timeout context for this specific publish operation
			publishCtx, publishCancel := context.WithTimeout(ctx, platformPublishTimeout)
			defer publishCancel()

			url, err := p.Publish(publishCtx, pst)
			resultCh <- domain.Result{
				Platform: p.Platform(),
				PostID:   extractPostID(url),
				URL:      url,
				Error:    err,
			}
		}(publisher, post)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultCh)

	// Collect results from channel
	for result := range resultCh {
		mu.Lock()
		results[result.Platform] = result
		mu.Unlock()
	}

	return results
}

// ValidatePublishers checks that all required platforms have valid publishers configured.
func (uc *PublishUseCase) ValidatePublishers(platforms []domain.Platform) error {
	var missing []domain.Platform
	for _, platform := range platforms {
		if _, ok := uc.publishers[platform]; !ok {
			missing = append(missing, platform)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing publishers for platforms: %v", missing)
	}
	return nil
}

// Default timeout for the entire publish operation across all platforms.
const defaultPublishTimeout = 5 * 60 * 1e9 // 5 minutes in nanoseconds

// Timeout for a single platform's publish operation.
const platformPublishTimeout = 2 * 60 * 1e9 // 2 minutes in nanoseconds

// extractPostID extracts the post ID from a URL.
// Platform-specific implementations may return just an ID or a full URL.
func extractPostID(url string) string {
	// If the URL contains the post ID, we return it as-is
	// Platform-specific adapters should return consistent formats
	return url
}
