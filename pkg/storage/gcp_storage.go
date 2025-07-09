package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPStorage struct {
	client    *storage.Client
	bucket    string
	cdnDomain string
}

func NewGCPStorage(projectID, bucket, credentialsFile, cdnDomain string) (*GCPStorage, error) {
	ctx := context.Background()

	var client *storage.Client
	var err error

	if credentialsFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	} else {
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
	}

	return &GCPStorage{
		client:    client,
		bucket:    bucket,
		cdnDomain: cdnDomain,
	}, nil
}

func (g *GCPStorage) Upload(ctx context.Context, request *UploadRequest) (*UploadResponse, error) {
	bucket := g.client.Bucket(g.bucket)
	object := bucket.Object(request.Key)

	writer := object.NewWriter(ctx)
	writer.ContentType = request.ContentType

	if len(request.Metadata) > 0 {
		writer.Metadata = request.Metadata
	}

	if request.CacheControl != "" {
		writer.CacheControl = request.CacheControl
	}

	size, err := io.Copy(writer, request.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write to GCP storage: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	url := g.generateURL(request.Key)

	return &UploadResponse{
		Key:  request.Key,
		URL:  url,
		Size: size,
	}, nil
}

func (g *GCPStorage) Download(ctx context.Context, key string) (*DownloadResponse, error) {
	bucket := g.client.Bucket(g.bucket)
	object := bucket.Object(key)

	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	attrs, err := object.Attrs(ctx)
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	return &DownloadResponse{
		Reader:       reader,
		Size:         attrs.Size,
		ContentType:  attrs.ContentType,
		Metadata:     attrs.Metadata,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
	}, nil
}

func (g *GCPStorage) Delete(ctx context.Context, key string) error {
	bucket := g.client.Bucket(g.bucket)
	object := bucket.Object(key)

	err := object.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete from GCP storage: %w", err)
	}

	return nil
}

func (g *GCPStorage) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	bucket := g.client.Bucket(g.bucket)
	bucket.Object(key)

	opts := &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	url, err := storage.SignedURL(g.bucket, key, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

func (g *GCPStorage) ListFiles(ctx context.Context, prefix string) ([]*FileInfo, error) {
	bucket := g.client.Bucket(g.bucket)

	query := &storage.Query{
		Prefix: prefix,
	}

	var files []*FileInfo

	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate objects: %w", err)
		}

		files = append(files, &FileInfo{
			Key:          attrs.Name,
			Size:         attrs.Size,
			ContentType:  attrs.ContentType,
			LastModified: attrs.Updated,
			ETag:         attrs.Etag,
			Metadata:     attrs.Metadata,
			URL:          g.generateURL(attrs.Name),
		})
	}

	return files, nil
}

func (g *GCPStorage) FileExists(ctx context.Context, key string) (bool, error) {
	bucket := g.client.Bucket(g.bucket)
	object := bucket.Object(key)

	_, err := object.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (g *GCPStorage) GetFileInfo(ctx context.Context, key string) (*FileInfo, error) {
	bucket := g.client.Bucket(g.bucket)
	object := bucket.Object(key)

	attrs, err := object.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         attrs.Size,
		ContentType:  attrs.ContentType,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
		Metadata:     attrs.Metadata,
		URL:          g.generateURL(key),
	}, nil
}

func (g *GCPStorage) generateURL(key string) string {
	if g.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", g.cdnDomain, key)
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucket, key)
}
