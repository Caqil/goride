package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type AWSS3Storage struct {
	client    *s3.Client
	bucket    string
	region    string
	cdnDomain string
}

func NewAWSS3Storage(region, bucket, cdnDomain string) (*AWSS3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSS3Storage{
		client:    s3.NewFromConfig(cfg),
		bucket:    bucket,
		region:    region,
		cdnDomain: cdnDomain,
	}, nil
}

func (a *AWSS3Storage) Upload(ctx context.Context, request *UploadRequest) (*UploadResponse, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(a.bucket),
		Key:         aws.String(request.Key),
		Body:        request.Reader,
		ContentType: aws.String(request.ContentType),
	}

	if request.Size > 0 {
		input.ContentLength = aws.Int64(request.Size)
	}

	if request.ACL != "" {
		input.ACL = types.ObjectCannedACL(request.ACL)
	}

	if request.CacheControl != "" {
		input.CacheControl = aws.String(request.CacheControl)
	}

	if len(request.Metadata) > 0 {
		input.Metadata = request.Metadata
	}

	resp, err := a.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	url := a.generateURL(request.Key)

	return &UploadResponse{
		Key:  request.Key,
		URL:  url,
		Size: request.Size,
		ETag: aws.ToString(resp.ETag),
	}, nil
}

func (a *AWSS3Storage) Download(ctx context.Context, key string) (*DownloadResponse, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	resp, err := a.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return &DownloadResponse{
		Reader:       resp.Body,
		Size:         aws.ToInt64(resp.ContentLength),
		ContentType:  aws.ToString(resp.ContentType),
		Metadata:     resp.Metadata,
		LastModified: aws.ToTime(resp.LastModified),
		ETag:         aws.ToString(resp.ETag),
	}, nil
}

func (a *AWSS3Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	_, err := a.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

func (a *AWSS3Storage) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(a.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	resp, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return resp.URL, nil
}

func (a *AWSS3Storage) ListFiles(ctx context.Context, prefix string) ([]*FileInfo, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
		Prefix: aws.String(prefix),
	}

	var files []*FileInfo

	paginator := s3.NewListObjectsV2Paginator(a.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			files = append(files, &FileInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
				ETag:         aws.ToString(obj.ETag),
				URL:          a.generateURL(aws.ToString(obj.Key)),
			})
		}
	}

	return files, nil
}

func (a *AWSS3Storage) FileExists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	_, err := a.client.HeadObject(ctx, input)
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (a *AWSS3Storage) GetFileInfo(ctx context.Context, key string) (*FileInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	resp, err := a.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         aws.ToInt64(resp.ContentLength),
		ContentType:  aws.ToString(resp.ContentType),
		LastModified: aws.ToTime(resp.LastModified),
		ETag:         aws.ToString(resp.ETag),
		Metadata:     resp.Metadata,
		URL:          a.generateURL(key),
	}, nil
}

func (a *AWSS3Storage) generateURL(key string) string {
	if a.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", a.cdnDomain, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", a.bucket, a.region, key)
}
