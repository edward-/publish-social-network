package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/edward-/publish-social-network/internal/domain"
)

// MockPublisher is a mock implementation of domain.Publisher for testing.
type MockPublisher struct {
	platform   domain.Platform
	publishErr error
	postID     string
	URL        string
	callCount  int
	mu         sync.Mutex
}

func (m *MockPublisher) Publish(ctx context.Context, post domain.Post) (string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.publishErr != nil {
		return "", m.publishErr
	}

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	return m.URL, nil
}

func (m *MockPublisher) Platform() domain.Platform {
	return m.platform
}

func (m *MockPublisher) ValidateConfig() error {
	return nil
}

func (m *MockPublisher) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// MockPost is a test post with valid data.
func MockPost() domain.Post {
	return domain.Post{
		Title:     "Test Title",
		Caption:   "Test Caption",
		MediaPath: "./test.mp4",
		MediaType: domain.MediaTypeVideo,
		Tags:      []string{"test", "golang"},
		Platforms: []domain.Platform{domain.Facebook, domain.YouTube},
		CreatedAt: time.Now(),
	}
}

func TestNewPublishUseCase(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}
	youtube := &MockPublisher{platform: domain.YouTube, URL: "https://youtube.com/watch?v=abc"}

	uc := NewPublishUseCase(facebook, youtube)

	if len(uc.publishers) != 2 {
		t.Errorf("expected 2 publishers, got %d", len(uc.publishers))
	}

	if uc.publishers[domain.Facebook] == nil {
		t.Error("expected Facebook publisher to be registered")
	}

	if uc.publishers[domain.YouTube] == nil {
		t.Error("expected YouTube publisher to be registered")
	}
}

func TestPublishUseCase_Publish_Success(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}
	youtube := &MockPublisher{platform: domain.YouTube, URL: "https://youtube.com/watch?v=abc"}

	uc := NewPublishUseCase(facebook, youtube)

	post := MockPost()
	ctx := context.Background()

	results := uc.Publish(ctx, post)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if !results[domain.Facebook].Success() {
		t.Errorf("Facebook publish failed: %v", results[domain.Facebook].Error)
	}

	if !results[domain.YouTube].Success() {
		t.Errorf("YouTube publish failed: %v", results[domain.YouTube].Error)
	}
}

func TestPublishUseCase_Publish_PartialFailure(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}
	youtube := &MockPublisher{platform: domain.YouTube, publishErr: errors.New("youtube API error")}

	uc := NewPublishUseCase(facebook, youtube)

	post := MockPost()
	ctx := context.Background()

	results := uc.Publish(ctx, post)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if !results[domain.Facebook].Success() {
		t.Errorf("Facebook publish should have succeeded")
	}

	if results[domain.YouTube].Success() {
		t.Errorf("YouTube publish should have failed")
	}

	if results[domain.YouTube].Error.Error() != "youtube API error" {
		t.Errorf("unexpected error message: %v", results[domain.YouTube].Error)
	}
}

func TestPublishUseCase_Publish_ValidationError(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}

	uc := NewPublishUseCase(facebook)

	// Post without caption should fail validation
	post := domain.Post{
		Title:     "Test Title",
		Caption:   "", // Missing caption
		MediaType: domain.MediaTypeText,
		Platforms: []domain.Platform{domain.Facebook},
	}

	ctx := context.Background()
	results := uc.Publish(ctx, post)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[domain.Facebook].Success() {
		t.Errorf("Facebook publish should have failed due to validation error")
	}
}

func TestPublishUseCase_Publish_UnsupportedPlatform(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}

	uc := NewPublishUseCase(facebook)

	post := domain.Post{
		Caption:   "Test Caption",
		MediaType: domain.MediaTypeText,
		Platforms: []domain.Platform{domain.TikTok}, // TikTok publisher not registered
	}

	ctx := context.Background()
	results := uc.Publish(ctx, post)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[domain.TikTok].Success() {
		t.Errorf("TikTok publish should have failed")
	}
}

func TestPublishUseCase_Publish_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	started := make(chan struct{})
	completed := make(chan struct{})

	facebook := &MockPublisher{
		platform: domain.Facebook,
		URL:      "https://facebook.com/post/123",
	}

	uc := NewPublishUseCase(facebook)

	post := MockPost()

	// Launch multiple concurrent publishes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started <- struct{}{}
			ctx := context.Background()
			uc.Publish(ctx, post)
			completed <- struct{}{}
		}()
	}

	// Wait for all goroutines to start
	for i := 0; i < 10; i++ {
		<-started
	}

	// Wait for all to complete
	wg.Wait()

	if facebook.GetCallCount() != 10 {
		t.Errorf("expected 10 calls to Facebook publisher, got %d", facebook.GetCallCount())
	}
}

func TestPublishUseCase_ValidatePublishers(t *testing.T) {
	facebook := &MockPublisher{platform: domain.Facebook, URL: "https://facebook.com/post/123"}
	youtube := &MockPublisher{platform: domain.YouTube, URL: "https://youtube.com/watch?v=abc"}

	uc := NewPublishUseCase(facebook, youtube)

	err := uc.ValidatePublishers([]domain.Platform{domain.Facebook, domain.YouTube})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = uc.ValidatePublishers([]domain.Platform{domain.Facebook, domain.TikTok})
	if err == nil {
		t.Error("expected error for missing TikTok publisher")
	}
}

func TestPost_Validate(t *testing.T) {
	tests := []struct {
		name    string
		post    domain.Post
		wantErr bool
	}{
		{
			name: "valid text post",
			post: domain.Post{
				Caption:   "Hello World",
				MediaType: domain.MediaTypeText,
				Platforms: []domain.Platform{domain.Facebook},
			},
			wantErr: false,
		},
		{
			name: "valid image post",
			post: domain.Post{
				Caption:     "Hello World",
				MediaPath:   "./image.jpg",
				MediaType:   domain.MediaTypeImage,
				Platforms:   []domain.Platform{domain.Facebook},
			},
			wantErr: false,
		},
		{
			name: "missing caption",
			post: domain.Post{
				Caption:   "",
				MediaType: domain.MediaTypeText,
				Platforms: []domain.Platform{domain.Facebook},
			},
			wantErr: true,
		},
		{
			name: "no platforms",
			post: domain.Post{
				Caption:   "Hello World",
				MediaType: domain.MediaTypeText,
				Platforms: []domain.Platform{},
			},
			wantErr: true,
		},
		{
			name: "video without media path",
			post: domain.Post{
				Caption:     "Hello World",
				MediaType:   domain.MediaTypeVideo,
				Platforms:   []domain.Platform{domain.YouTube},
			},
			wantErr: true,
		},
		{
			name: "youtube without title",
			post: domain.Post{
				Caption:     "Hello World",
				MediaPath:   "./video.mp4",
				MediaType:   domain.MediaTypeVideo,
				Platforms:   []domain.Platform{domain.YouTube},
			},
			wantErr: true,
		},
		{
			name: "valid youtube post",
			post: domain.Post{
				Title:       "My Video",
				Caption:     "Hello World",
				MediaPath:   "./video.mp4",
				MediaType:   domain.MediaTypeVideo,
				Platforms:   []domain.Platform{domain.YouTube},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.post.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("post.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
