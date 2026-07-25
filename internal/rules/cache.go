package rules

import (
	"context"
	"io/fs"
	"sync"
)

type CacheKey string

type Cache struct {
	mu   sync.RWMutex
	data map[CacheKey]interface{}
	fs   fs.FS
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[CacheKey]interface{}),
	}
}

func NewCacheWithFS(fileSystem fs.FS) *Cache {
	return &Cache{
		data: make(map[CacheKey]interface{}),
		fs:   fileSystem,
	}
}

func (c *Cache) FS() fs.FS {
	return c.fs
}

func (c *Cache) Get(key CacheKey) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *Cache) Set(key CacheKey, val interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = val
}

type cacheCtxKey struct{}

func WithCache(ctx context.Context, c *Cache) context.Context {
	return context.WithValue(ctx, cacheCtxKey{}, c)
}

func CacheFromContext(ctx context.Context) *Cache {
	if c, ok := ctx.Value(cacheCtxKey{}).(*Cache); ok {
		return c
	}
	return nil
}
