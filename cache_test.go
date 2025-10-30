package zendia

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type TestEntity struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	TenantID uuid.UUID `json:"tenant_id"`
}

func (e *TestEntity) GetID() uuid.UUID         { return e.ID }
func (e *TestEntity) SetID(id uuid.UUID)      { e.ID = id }
func (e *TestEntity) SetTenantID(s string)    { e.TenantID = uuid.MustParse(s) }

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

	// Test Set and Get
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

func TestCachedRepository(t *testing.T) {
	// Create base repository (in-memory)
	baseRepo := NewMemoryRepository[*TestEntity, uuid.UUID](func() uuid.UUID {
		return uuid.New()
	})

	// Create cache
	cache := NewMemoryCache(MemoryCacheConfig{
		CacheConfig: CacheConfig{
			TTL: 5 * time.Minute,
		},
		MaxSize: 100,
	})

	// Create cached repository
	cachedRepo := NewCachedRepository(baseRepo, cache, CacheConfig{
		TTL: 5 * time.Minute,
	}, "TestEntity")

	ctx := context.Background()

	// Create entity
	entity := &TestEntity{
		Name: "Test Entity",
	}

	created, err := cachedRepo.Create(ctx, entity)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// First get - should hit database
	result1, err := cachedRepo.GetByID(ctx, created.GetID())
	if err != nil {
		t.Fatalf("Failed to get entity: %v", err)
	}

	if result1.Name != "Test Entity" {
		t.Fatalf("Expected 'Test Entity', got %s", result1.Name)
	}

	// Second get - should hit cache
	result2, err := cachedRepo.GetByID(ctx, created.GetID())
	if err != nil {
		t.Fatalf("Failed to get entity from cache: %v", err)
	}

	if result2.Name != "Test Entity" {
		t.Fatalf("Expected 'Test Entity', got %s", result2.Name)
	}

	// Update should invalidate cache
	result1.Name = "Updated Entity"
	updated, err := cachedRepo.Update(ctx, created.GetID(), result1)
	if err != nil {
		t.Fatalf("Failed to update entity: %v", err)
	}

	if updated.Name != "Updated Entity" {
		t.Fatalf("Expected 'Updated Entity', got %s", updated.Name)
	}
}