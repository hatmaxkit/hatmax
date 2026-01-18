package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hatmaxkit/hatmax/image"
)

// Store implements image.Store using S3-compatible storage.
type Store struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

// Options configures the S3 store.
type Options struct {
	Bucket    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	BaseURL   string
}

// NewStore creates a new S3 store.
func NewStore(ctx context.Context, opts Options) (*Store, error) {
	if opts.Bucket == "" {
		return nil, fmt.Errorf("s3: bucket is required")
	}
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("s3: base_url is required")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(opts.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			opts.AccessKey,
			opts.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: cannot load config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
			o.UsePathStyle = true
		}
	})

	return &Store{
		client:  client,
		bucket:  opts.Bucket,
		baseURL: strings.TrimSuffix(opts.BaseURL, "/"),
	}, nil
}

// Put stores data at the given path.
func (s *Store) Put(ctx context.Context, path string, data io.Reader) error {
	buf, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("s3: cannot read data: %w", err)
	}

	contentType := http.DetectContentType(buf)

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(path),
		Body:        strings.NewReader(string(buf)),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3: cannot put object: %w", err)
	}

	return nil
}

// Get returns a reader for the file at the given path.
func (s *Store) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: cannot get object: %w", err)
	}

	return result.Body, nil
}

// Delete removes the file at the given path.
func (s *Store) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("s3: cannot delete object: %w", err)
	}

	return nil
}

// URL returns a servable URL for the image.
func (s *Store) URL(path string) string {
	return s.baseURL + "/" + path
}

var _ image.Store = (*Store)(nil)
