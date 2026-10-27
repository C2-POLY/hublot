// SPDX-License-Identifier: AGPL-3.0-or-later

// Package storage keeps attachments, which GitHub cannot take through its API (PRD §13).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// Presigner is implemented by stores that can hand out a short-lived direct URL.
type Presigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration, filename string) (string, error)
}

var ErrNotFound = errors.New("storage: not found")

// FS stores objects as files under a directory. Fine for a single instance.
type FS struct{ Dir string }

func (f FS) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if strings.Contains(key, "..") || clean == "/" {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return filepath.Join(f.Dir, clean), nil
}

func (f FS) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (f FS) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := f.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return file, err
}

func (f FS) Delete(_ context.Context, key string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// S3 stores objects in any S3-compatible bucket (OVH Object Storage, R2, MinIO…).
type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(endpoint, region, bucket, accessKey, secretKey string, useSSL bool) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	return &S3{client: client, bucket: bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	// With an unknown size, minio-go otherwise buffers one huge part per upload.
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType, PartSize: 5 << 20})
	return err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration, filename string) (string, error) {
	params := url.Values{}
	params.Set("response-content-disposition", "inline; filename*=UTF-8''"+url.PathEscape(filename))
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
