package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStorage struct {
	basePath string
	baseURL  string
}

func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	err := os.MkdirAll(basePath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

func (l *LocalStorage) Upload(ctx context.Context, request *UploadRequest) (*UploadResponse, error) {
	filePath := filepath.Join(l.basePath, request.Key)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	size, err := io.Copy(file, request.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Generate URL
	url := l.generateURL(request.Key)

	return &UploadResponse{
		Key:      request.Key,
		URL:      url,
		Size:     size,
		Location: filePath,
	}, nil
}

func (l *LocalStorage) Download(ctx context.Context, key string) (*DownloadResponse, error) {
	filePath := filepath.Join(l.basePath, key)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &DownloadResponse{
		Reader:       file,
		Size:         stat.Size(),
		ContentType:  l.getContentType(key),
		LastModified: stat.ModTime(),
	}, nil
}

func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(l.basePath, key)
	return os.Remove(filePath)
}

func (l *LocalStorage) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// Local storage doesn't support expiring URLs
	return l.generateURL(key), nil
}

func (l *LocalStorage) ListFiles(ctx context.Context, prefix string) ([]*FileInfo, error) {
	var files []*FileInfo

	prefixPath := filepath.Join(l.basePath, prefix)

	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasPrefix(path, prefixPath) {
			return nil
		}

		relPath, err := filepath.Rel(l.basePath, path)
		if err != nil {
			return err
		}

		files = append(files, &FileInfo{
			Key:          filepath.ToSlash(relPath),
			Size:         info.Size(),
			ContentType:  l.getContentType(relPath),
			LastModified: info.ModTime(),
			URL:          l.generateURL(filepath.ToSlash(relPath)),
		})

		return nil
	})

	return files, err
}

func (l *LocalStorage) FileExists(ctx context.Context, key string) (bool, error) {
	filePath := filepath.Join(l.basePath, key)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l *LocalStorage) GetFileInfo(ctx context.Context, key string) (*FileInfo, error) {
	filePath := filepath.Join(l.basePath, key)

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  l.getContentType(key),
		LastModified: stat.ModTime(),
		URL:          l.generateURL(key),
	}, nil
}

func (l *LocalStorage) generateURL(key string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(l.baseURL, "/"), key)
}

func (l *LocalStorage) getContentType(key string) string {
	ext := strings.ToLower(filepath.Ext(key))

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	return "application/octet-stream"
}
