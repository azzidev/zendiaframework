package zendia

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(MemoryCacheConfig{
		CacheConfig: CacheConfig{
			TTL: 1 * time.Second,
		},
		MaxSize: 100,
	})

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	err := cache.Set(ctx, key, value, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	result, found := cache.Get(ctx, key)
	if !found {
		t.Fatal("Cache miss when should hit")
	}

	if string(result) != string(value) {
		t.Fatalf("Expected %s, got %s", value, result)
	}

	// Test expiration
	time.Sleep(1100 * time.Millisecond)
	_, found = cache.Get(ctx, key)
	if found {
		t.Fatal("Cache hit when should miss (expired)")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(MemoryCacheConfig{
		CacheConfig: CacheConfig{TTL: 5 * time.Minute},
		MaxSize:     100,
	})

	ctx := context.Background()
	cache.Set(ctx, "key1", []byte("value1"), 0)

	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Fatal("Should find key1")
	}

	cache.Delete(ctx, "key1")

	_, found = cache.Get(ctx, "key1")
	if found {
		t.Fatal("Should not find key1 after delete")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(MemoryCacheConfig{
		CacheConfig: CacheConfig{TTL: 5 * time.Minute},
		MaxSize:     100,
	})

	ctx := context.Background()
	cache.Set(ctx, "key1", []byte("value1"), 0)
	cache.Set(ctx, "key2", []byte("value2"), 0)

	cache.Clear(ctx)

	_, found1 := cache.Get(ctx, "key1")
	_, found2 := cache.Get(ctx, "key2")

	if found1 || found2 {
		t.Fatal("Should not find any keys after clear")
	}
}
