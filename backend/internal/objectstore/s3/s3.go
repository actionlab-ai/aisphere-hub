package s3store

import (
	"context"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Endpoint, Region, AccessKey, SecretKey, Bucket string
	UseSSL                                         bool
	Prefix                                         string
}
type Store struct {
	client         *minio.Client
	bucket, prefix string
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	ep := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "http://"), "https://")
	cli, err := minio.New(ep, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: cfg.UseSSL, Region: cfg.Region})
	if err != nil {
		return nil, err
	}
	ok, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := cli.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, err
		}
	}
	return &Store{client: cli, bucket: cfg.Bucket, prefix: strings.Trim(cfg.Prefix, "/")}, nil
}
func (s *Store) key(k string) string {
	k = strings.Trim(k, "/")
	if s.prefix == "" {
		return k
	}
	return s.prefix + "/" + k
}
func (s *Store) Put(ctx context.Context, k string, r io.Reader, o ports.PutOptions) (*ports.ObjectInfo, error) {
	key := s.key(k)
	info, err := s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{ContentType: o.ContentType, UserMetadata: o.Metadata})
	if err != nil {
		return nil, err
	}
	et := strings.Trim(info.ETag, "\"")
	return &ports.ObjectInfo{Key: k, Size: info.Size, ETag: et, ContentMD5: et, ContentType: o.ContentType, UpdatedAt: time.Now()}, nil
}
func (s *Store) Get(ctx context.Context, k string) (io.ReadCloser, *ports.ObjectInfo, error) {
	key := s.key(k)
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	st, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, nil, err
	}
	et := strings.Trim(st.ETag, "\"")
	return obj, &ports.ObjectInfo{Key: k, Size: st.Size, ETag: et, ContentMD5: et, ContentType: st.ContentType, UpdatedAt: st.LastModified}, nil
}
func (s *Store) Delete(ctx context.Context, k string) error {
	return s.client.RemoveObject(ctx, s.bucket, s.key(k), minio.RemoveObjectOptions{})
}
func (s *Store) Exists(ctx context.Context, k string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, s.key(k), minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
		return false, nil
	}
	return false, err
}
func (s *Store) PresignGet(ctx context.Context, k string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, s.key(k), ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

var _ ports.ObjectStore = (*Store)(nil)
