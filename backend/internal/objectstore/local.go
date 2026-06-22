package objectstore

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/ports"
)

type LocalStore struct{ Root string }

func NewLocal(root string) *LocalStore {
	if root == "" {
		root = "./data/objects"
	}
	return &LocalStore{Root: root}
}
func (s *LocalStore) Put(ctx context.Context, key string, reader io.Reader, opts ports.PutOptions) (*ports.ObjectInfo, error) {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	h := md5.New()
	n, err := io.Copy(io.MultiWriter(f, h), reader)
	cerr := f.Close()
	if err != nil {
		return nil, err
	}
	if cerr != nil {
		return nil, cerr
	}
	if err := os.Rename(tmp, p); err != nil {
		return nil, err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return &ports.ObjectInfo{Key: key, Size: n, ETag: sum, ContentMD5: sum, ContentType: opts.ContentType, UpdatedAt: time.Now()}, nil
}
func (s *LocalStore) Get(ctx context.Context, key string) (io.ReadCloser, *ports.ObjectInfo, error) {
	p := s.path(key)
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	st, _ := f.Stat()
	return f, &ports.ObjectInfo{Key: key, Size: st.Size(), UpdatedAt: st.ModTime()}, nil
}
func (s *LocalStore) Delete(ctx context.Context, key string) error { return os.Remove(s.path(key)) }
func (s *LocalStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := os.Stat(s.path(key))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
func (s *LocalStore) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return "file://" + s.path(key), nil
}
func (s *LocalStore) path(key string) string {
	key = strings.TrimPrefix(filepath.Clean("/"+key), "/")
	return filepath.Join(s.Root, key)
}
