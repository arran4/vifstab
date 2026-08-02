package rules

import (
	"context"
	"sync"
	"testing"
)

func TestNewCache(t *testing.T) {
	c := NewCache()
	if c == nil {
		t.Fatal("NewCache() returned nil")
	}
	if c.data == nil {
		t.Error("NewCache() did not initialize data map")
	}
}

func TestCache_GetSet(t *testing.T) {
	c := NewCache()

	// Test cache miss
	val, ok := c.Get("test_key")
	if ok {
		t.Errorf("Expected cache miss for non-existent key, got ok=true, val=%v", val)
	}

	// Test cache hit
	c.Set("test_key", "test_value")
	val, ok = c.Get("test_key")
	if !ok {
		t.Error("Expected cache hit for existing key, got ok=false")
	}
	if val != "test_value" {
		t.Errorf("Expected value 'test_value', got %v", val)
	}

	// Test update existing key
	c.Set("test_key", "new_value")
	val, ok = c.Get("test_key")
	if !ok {
		t.Error("Expected cache hit for existing key after update")
	}
	if val != "new_value" {
		t.Errorf("Expected value 'new_value', got %v", val)
	}
}

func TestCache_Concurrency(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup

	// Concurrent writes and reads
	for i := 0; i < 1000; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.Set(CacheKey("key"), i)
		}(i)
		go func(i int) {
			defer wg.Done()
			c.Get(CacheKey("key"))
		}(i)
	}

	wg.Wait()
}

func TestWithCacheAndCacheFromContext(t *testing.T) {
	c := NewCache()
	ctx := context.Background()

	// Test CacheFromContext with no cache in context
	got := CacheFromContext(ctx)
	if got != nil {
		t.Errorf("Expected nil cache from empty context, got %v", got)
	}

	// Test WithCache and CacheFromContext
	ctxWithCache := WithCache(ctx, c)
	got = CacheFromContext(ctxWithCache)
	if got != c {
		t.Errorf("Expected cache %p from context, got %p", c, got)
	}
}
