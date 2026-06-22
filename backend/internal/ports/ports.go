package ports

import (
	"context"
	"io"
	"time"
)

// Repository is intentionally resource-oriented rather than skill-only.
// It mirrors the Nacos ai_resource / ai_resource_version style while keeping
// the implementation replaceable: mysql, memory, sqlite, etc.
type Repository interface {
	ResourceRepository
	ResourceVersionRepository
	GroupRepository
	AuditRepository
}

type ResourceRepository interface{}
type ResourceVersionRepository interface{}
type GroupRepository interface{}
type AuditRepository interface{}

// Cache is the low level cache port. Redis single node, Redis Cluster,
// in-memory cache and noop cache should all implement this small contract.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// RuntimeCache is the higher-level cache contract used by the skill runtime.
// Keep route and version metadata separated so label changes can invalidate
// routing without touching immutable version content.
type RuntimeCache interface {
	GetRoute(ctx context.Context, namespaceID, resourceType, name, label string) (version string, ok bool, err error)
	SetRoute(ctx context.Context, namespaceID, resourceType, name, label, version string, ttl time.Duration) error
	DeleteRoutes(ctx context.Context, namespaceID, resourceType, name string, labels ...string) error

	GetVersionMeta(ctx context.Context, namespaceID, resourceType, name, version string, out any) (ok bool, err error)
	SetVersionMeta(ctx context.Context, namespaceID, resourceType, name, version string, value any, ttl time.Duration) error
	DeleteVersionMeta(ctx context.Context, namespaceID, resourceType, name string, versions ...string) error

	GetGroupManifest(ctx context.Context, namespaceID, groupName, label string, out any) (ok bool, err error)
	SetGroupManifest(ctx context.Context, namespaceID, groupName, label string, value any, ttl time.Duration) error
	DeleteGroupManifests(ctx context.Context, namespaceID, groupName string) error

	IncrementDownload(ctx context.Context, namespaceID, resourceType, name, version string, delta int64) error
}

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

type ObjectInfo struct {
	Key         string
	Size        int64
	ETag        string
	ContentMD5  string
	ContentType string
	UpdatedAt   time.Time
}

// ObjectStore is the file-content port. Production uses MinIO/S3; dev/test can
// use localfs or memory. Business code should never import a concrete SDK.
type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, opts PutOptions) (*ObjectInfo, error)
	Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}
