package zendia

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheProvider interface comum para diferentes implementações de cache
type CacheProvider interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// CacheConfig configuração básica do cache
type CacheConfig struct {
	TTL       time.Duration
	KeyPrefix string
}

// MemoryCacheConfig configuração específica do cache em memória
type MemoryCacheConfig struct {
	CacheConfig
	MaxSize   int
	MaxMemory int64 // bytes
}

// cacheItem item do cache em memória
type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

// MemoryCache implementação de cache em memória
type MemoryCache struct {
	config MemoryCacheConfig
	items  sync.Map
	size   int64
	mutex  sync.RWMutex
}

// NewMemoryCache cria um novo cache em memória
func NewMemoryCache(config MemoryCacheConfig) *MemoryCache {
	if config.TTL == 0 {
		config.TTL = 10 * time.Minute
	}
	if config.MaxSize == 0 {
		config.MaxSize = 10000
	}
	if config.MaxMemory == 0 {
		config.MaxMemory = 5 * 1024 * 1024 // 5MB
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "zendia:"
	}

	cache := &MemoryCache{
		config: config,
	}

	// Cleanup goroutine
	go cache.cleanup()

	return cache
}

func (mc *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
	fullKey := mc.config.KeyPrefix + key

	if item, ok := mc.items.Load(fullKey); ok {
		cacheItem := item.(*cacheItem)
		if time.Now().Before(cacheItem.expiresAt) {
			return cacheItem.data, true
		}
		mc.items.Delete(fullKey)
	}
	return nil, false
}

func (mc *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = mc.config.TTL
	}

	fullKey := mc.config.KeyPrefix + key
	item := &cacheItem{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Check memory limit
	if mc.size+int64(len(value)) > mc.config.MaxMemory {
		mc.evictOldest()
	}

	mc.items.Store(fullKey, item)
	mc.size += int64(len(value))

	return nil
}

func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	fullKey := mc.config.KeyPrefix + key
	if item, ok := mc.items.LoadAndDelete(fullKey); ok {
		mc.mutex.Lock()
		mc.size -= int64(len(item.(*cacheItem).data))
		mc.mutex.Unlock()
	}
	return nil
}

func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.items = sync.Map{}
	mc.size = 0
	return nil
}

func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		mc.items.Range(func(key, value interface{}) bool {
			item := value.(*cacheItem)
			if now.After(item.expiresAt) {
				mc.items.Delete(key)
				mc.mutex.Lock()
				mc.size -= int64(len(item.data))
				mc.mutex.Unlock()
			}
			return true
		})
	}
}

func (mc *MemoryCache) evictOldest() {
	// Simple eviction - remove first expired item found
	now := time.Now()
	mc.items.Range(func(key, value interface{}) bool {
		item := value.(*cacheItem)
		if now.After(item.expiresAt) {
			mc.items.Delete(key)
			mc.size -= int64(len(item.data))
			return false // Stop after first eviction
		}
		return true
	})
}

// CachedRepository wrapper que adiciona cache a qualquer repository
type CachedRepository[T any, ID comparable] struct {
	base     Repository[T, ID]
	cache    CacheProvider
	config   CacheConfig
	typeName string
}

// NewCachedRepository cria um repository com cache
func NewCachedRepository[T any, ID comparable](base Repository[T, ID], cache CacheProvider, config CacheConfig, typeName string) *CachedRepository[T, ID] {
	if config.TTL == 0 {
		config.TTL = 10 * time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "zendia:"
	}

	return &CachedRepository[T, ID]{
		base:     base,
		cache:    cache,
		config:   config,
		typeName: typeName,
	}
}

func (cr *CachedRepository[T, ID]) makeKey(operation string, id ID) string {
	return fmt.Sprintf("%s:%s:%v", cr.typeName, operation, id)
}

func (cr *CachedRepository[T, ID]) makeTenantKey(operation string, tenantID string) string {
	return fmt.Sprintf("%s:%s:tenant:%s", cr.typeName, operation, tenantID)
}

func (cr *CachedRepository[T, ID]) GetByID(ctx context.Context, id ID) (T, error) {
	var zero T
	key := cr.makeKey("get", id)

	// Try cache first
	if data, found := cr.cache.Get(ctx, key); found {
		var result T
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	// Cache miss - get from base repository
	result, err := cr.base.GetByID(ctx, id)
	if err != nil {
		return zero, err
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		cr.cache.Set(ctx, key, data, cr.config.TTL)
	}

	return result, nil
}

func (cr *CachedRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	result, err := cr.base.Create(ctx, entity)
	if err != nil {
		return result, err
	}

	// Invalidate tenant cache
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantKey := cr.makeTenantKey("list", tenantInfo.TenantID)
		cr.cache.Delete(ctx, tenantKey)
	}

	return result, nil
}

func (cr *CachedRepository[T, ID]) Update(ctx context.Context, id ID, entity T) (T, error) {
	result, err := cr.base.Update(ctx, id, entity)
	if err != nil {
		return result, err
	}

	// Invalidate specific item cache
	key := cr.makeKey("get", id)
	cr.cache.Delete(ctx, key)

	// Invalidate tenant cache
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantKey := cr.makeTenantKey("list", tenantInfo.TenantID)
		cr.cache.Delete(ctx, tenantKey)
	}

	return result, nil
}

func (cr *CachedRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	err := cr.base.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate specific item cache
	key := cr.makeKey("get", id)
	cr.cache.Delete(ctx, key)

	// Invalidate tenant cache
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantKey := cr.makeTenantKey("list", tenantInfo.TenantID)
		cr.cache.Delete(ctx, tenantKey)
	}

	return nil
}

func (cr *CachedRepository[T, ID]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// For GetAll, we only cache by tenant to keep it simple
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID == "" {
		// No tenant context, don't cache
		return cr.base.GetAll(ctx, filters)
	}

	key := cr.makeTenantKey("list", tenantInfo.TenantID)

	// Try cache first
	if data, found := cr.cache.Get(ctx, key); found {
		var result []T
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	// Cache miss - get from base repository
	result, err := cr.base.GetAll(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		cr.cache.Set(ctx, key, data, cr.config.TTL)
	}

	return result, nil
}

func (cr *CachedRepository[T, ID]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	return cr.GetAll(ctx, filters)
}
