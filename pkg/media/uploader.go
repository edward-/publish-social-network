// Package media provides shared utilities for handling media files.
package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Validator validates media files before upload.
type Validator struct {
	maxImageSize int64
	maxVideoSize int64
}

// NewValidator creates a new media validator with default limits.
func NewValidator() *Validator {
	return &Validator{
		maxImageSize: 100 * 1024 * 1024, // 100MB
		maxVideoSize: 2 * 1024 * 1024 * 1024, // 2GB
	}
}

// MediaInfo contains information about a media file.
type MediaInfo struct {
	Path      string
	Size      int64
	MIMEType  string
	Extension string
}

// ReadAndValidate reads a file from disk and validates it.
func (v *Validator) ReadAndValidate(path string) (*MediaInfo, io.ReadCloser, error) {
	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("not a regular file: %s", path)
	}

	// Get file extension
	ext := filepath.Ext(path)
	lowerExt := ext
	if len(ext) > 0 {
		lowerExt = ext[1:] // Remove leading dot
	}

	// Determine MIME type from extension
	mimeType := v.getMIMEType(lowerExt)

	// Validate file size based on type
	if v.isImage(lowerExt) && info.Size() > v.maxImageSize {
		return nil, nil, fmt.Errorf("image file too large: %d bytes (max: %d)", info.Size(), v.maxImageSize)
	}
	if v.isVideo(lowerExt) && info.Size() > v.maxVideoSize {
		return nil, nil, fmt.Errorf("video file too large: %d bytes (max: %d)", info.Size(), v.maxVideoSize)
	}

	// Open file for reading
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &MediaInfo{
		Path:      path,
		Size:      info.Size(),
		MIMEType:  mimeType,
		Extension: lowerExt,
	}, file, nil
}

// ReadFile reads the entire contents of a file.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

// getMIMEType returns the MIME type based on file extension.
func (v *Validator) getMIMEType(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "avi":
		return "video/x-msvideo"
	case "mkv":
		return "video/x-matroska"
	case "webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// isImage returns true if the extension is a supported image type.
func (v *Validator) isImage(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp":
		return true
	}
	return false
}

// isVideo returns true if the extension is a supported video type.
func (v *Validator) isVideo(ext string) bool {
	switch ext {
	case "mp4", "mov", "avi", "mkv", "webm":
		return true
	}
	return false
}

// IsImage checks if a path points to an image file.
func IsImage(path string) bool {
	ext := filepath.Ext(path)
	if len(ext) > 0 {
		ext = ext[1:]
	}
	v := NewValidator()
	return v.isImage(ext)
}

// IsVideo checks if a path points to a video file.
func IsVideo(path string) bool {
	ext := filepath.Ext(path)
	if len(ext) > 0 {
		ext = ext[1:]
	}
	v := NewValidator()
	return v.isVideo(ext)
}
