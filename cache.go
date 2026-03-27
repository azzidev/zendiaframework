package zendia

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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
	MaxMemory int64
}

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
		config.MaxMemory = 5 * 1024 * 1024
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "zendia:"
	}

	cache := &MemoryCache{config: config}
	go cache.cleanup()
	return cache
}

func (mc *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
	fullKey := mc.config.KeyPrefix + key
	if item, ok := mc.items.Load(fullKey); ok {
		ci := item.(*cacheItem)
		if time.Now().Before(ci.expiresAt) {
			return ci.data, true
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
	now := time.Now()
	mc.items.Range(func(key, value interface{}) bool {
		item := value.(*cacheItem)
		if now.After(item.expiresAt) {
			mc.items.Delete(key)
			mc.size -= int64(len(item.data))
			return false
		}
		return true
	})
}

// CachedRepository wrapper que adiciona cache ao Repository
type CachedRepository[T MongoAuditableEntity] struct {
	base     *Repository[T]
	cache    CacheProvider
	config   CacheConfig
	typeName string
}

// NewCachedRepository cria um repository com cache
func NewCachedRepository[T MongoAuditableEntity](base *Repository[T], cache CacheProvider, config CacheConfig, typeName string) *CachedRepository[T] {
	if config.TTL == 0 {
		config.TTL = 10 * time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "zendia:"
	}

	return &CachedRepository[T]{
		base:     base,
		cache:    cache,
		config:   config,
		typeName: typeName,
	}
}

func (cr *CachedRepository[T]) makeKey(operation string, id uuid.UUID) string {
	return fmt.Sprintf("%s:%s:%v", cr.typeName, operation, id)
}

func (cr *CachedRepository[T]) makeTenantKey(operation string, tenantID string) string {
	return fmt.Sprintf("%s:%s:tenant:%s", cr.typeName, operation, tenantID)
}

func (cr *CachedRepository[T]) Create(ctx context.Context, entity T) (T, error) {
	result, err := cr.base.Create(ctx, entity)
	if err != nil {
		return result, err
	}

	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		cr.cache.Delete(ctx, cr.makeTenantKey("list", tenantInfo.TenantID))
	}

	return result, nil
}

func (cr *CachedRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	var zero T
	key := cr.makeKey("get", id)

	if data, found := cr.cache.Get(ctx, key); found {
		var result T
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	result, err := cr.base.GetByID(ctx, id)
	if err != nil {
		return zero, err
	}

	if data, err := json.Marshal(result); err == nil {
		cr.cache.Set(ctx, key, data, cr.config.TTL)
	}

	return result, nil
}

func (cr *CachedRepository[T]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	return cr.base.GetFirst(ctx, filters)
}

func (cr *CachedRepository[T]) Update(ctx context.Context, id uuid.UUID, entity T) (T, error) {
	result, err := cr.base.Update(ctx, id, entity)
	if err != nil {
		return result, err
	}

	cr.cache.Delete(ctx, cr.makeKey("get", id))

	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		cr.cache.Delete(ctx, cr.makeTenantKey("list", tenantInfo.TenantID))
	}

	return result, nil
}

func (cr *CachedRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	err := cr.base.Delete(ctx, id)
	if err != nil {
		return err
	}

	cr.cache.Delete(ctx, cr.makeKey("get", id))

	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		cr.cache.Delete(ctx, cr.makeTenantKey("list", tenantInfo.TenantID))
	}

	return nil
}

func (cr *CachedRepository[T]) GetAll(ctx context.Context, filters map[string]interface{}, opts ...*QueryOptions) ([]T, error) {
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID == "" || len(filters) > 0 {
		return cr.base.GetAll(ctx, filters, opts...)
	}

	key := cr.makeTenantKey("list", tenantInfo.TenantID)

	if data, found := cr.cache.Get(ctx, key); found {
		var result []T
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
	}

	result, err := cr.base.GetAll(ctx, filters, opts...)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(result); err == nil {
		cr.cache.Set(ctx, key, data, cr.config.TTL)
	}

	return result, nil
}

func (cr *CachedRepository[T]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int, opts ...*QueryOptions) ([]T, error) {
	return cr.base.GetAllSkipTake(ctx, filters, skip, take, opts...)
}

func (cr *CachedRepository[T]) List(ctx context.Context, filters map[string]interface{}, opts ...*QueryOptions) ([]T, error) {
	return cr.GetAll(ctx, filters, opts...)
}

func (cr *CachedRepository[T]) GetHistory(ctx context.Context, entityID uuid.UUID) ([]HistoryEntry, error) {
	return cr.base.GetHistory(ctx, entityID)
}

func (cr *CachedRepository[T]) Aggregate(ctx context.Context, pipeline []interface{}) ([]T, error) {
	return cr.base.Aggregate(ctx, pipeline)
}

func (cr *CachedRepository[T]) AggregateRaw(ctx context.Context, pipeline []interface{}) ([]map[string]interface{}, error) {
	return cr.base.AggregateRaw(ctx, pipeline)
}
