package zendia

import (
	"context"
	"time"
)

// RedisClient interface para compatibilidade com diferentes clientes Redis
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	FlushAll(ctx context.Context) error
}

// RedisCacheConfig configuração específica do cache Redis
type RedisCacheConfig struct {
	CacheConfig
	Client RedisClient
}

// RedisCache implementação de cache usando Redis
type RedisCache struct {
	config RedisCacheConfig
}

// NewRedisCache cria um novo cache Redis
func NewRedisCache(config RedisCacheConfig) *RedisCache {
	if config.TTL == 0 {
		config.TTL = 10 * time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "zendia:"
	}

	return &RedisCache{
		config: config,
	}
}

func (rc *RedisCache) Get(ctx context.Context, key string) ([]byte, bool) {
	fullKey := rc.config.KeyPrefix + key
	
	result, err := rc.config.Client.Get(ctx, fullKey)
	if err != nil {
		return nil, false
	}
	
	return []byte(result), true
}

func (rc *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = rc.config.TTL
	}

	fullKey := rc.config.KeyPrefix + key
	return rc.config.Client.Set(ctx, fullKey, value, ttl)
}

func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.config.KeyPrefix + key
	return rc.config.Client.Del(ctx, fullKey)
}

func (rc *RedisCache) Clear(ctx context.Context) error {
	return rc.config.Client.FlushAll(ctx)
}