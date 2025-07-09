
package storage

import (
	"context"
	"io"
	"time"
)

type StorageProvider interface {
	Upload(ctx context.Context, request *UploadRequest) (*UploadResponse, error)
	Download(ctx context.Context, key string) (*DownloadResponse, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	ListFiles(ctx context.Context, prefix string) ([]*FileInfo, error)
	FileExists(ctx context.Context, key string) (bool, error)
	GetFileInfo(ctx context.Context, key string) (*FileInfo, error)
}

type UploadRequest struct {
	Key         string            `json:"key"`
	Reader      io.Reader         `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata"`
	ACL         string            `json:"acl"`
	CacheControl string           `json:"cache_control"`
}

type UploadResponse struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	ETag     string `json:"etag"`
	Location string `json:"location"`
}

type DownloadResponse struct {
	Reader      io.ReadCloser     `json:"-"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	LastModified time.Time        `json:"last_modified"`
	ETag        string            `json:"etag"`
}

type FileInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata"`
	URL          string            `json:"url"`
}