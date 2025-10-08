package zendia

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}

func (u *TestUser) SetCreatedAt(t time.Time) { u.CreatedAt = t }
func (u *TestUser) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }
func (u *TestUser) SetCreatedBy(s string)    { u.CreatedBy = s }
func (u *TestUser) SetUpdatedBy(s string)    { u.UpdatedBy = s }
func (u *TestUser) SetTenantID(s string)     { u.TenantID = s }

func TestMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository[*TestUser, int](func() int { return 1 })
	ctx := context.Background()

	user := &TestUser{Name: "João"}
	
	// Test Create
	created, err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.Equal(t, "João", created.Name)

	// Test GetByID
	found, err := repo.GetByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, "João", found.Name)

	// Test Update
	found.Name = "João Silva"
	updated, err := repo.Update(ctx, 1, found)
	assert.NoError(t, err)
	assert.Equal(t, "João Silva", updated.Name)

	// Test List
	users, err := repo.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, users, 1)

	// Test Delete
	err = repo.Delete(ctx, 1)
	assert.NoError(t, err)

	// Test GetByID after delete
	_, err = repo.GetByID(ctx, 1)
	assert.Error(t, err)
}

func TestAuditRepository(t *testing.T) {
	baseRepo := NewMemoryRepository[*TestUser, int](func() int { return 1 })
	auditRepo := NewAuditRepository[*TestUser, int](baseRepo)
	
	// Cria contexto com tenant info
	ctx := context.WithValue(context.Background(), TenantIDKey, "test-tenant")
	ctx = context.WithValue(ctx, UserIDKey, "test-user")
	ctx = context.WithValue(ctx, ActionAtKey, time.Now())

	user := &TestUser{Name: "João"}
	
	// Test Create with audit (sem tenant context)
	created, err := auditRepo.Create(ctx, user)
	assert.NoError(t, err)
	assert.Equal(t, "João", created.Name)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	// Test Update with audit
	time.Sleep(1 * time.Millisecond) // Garante diferença de tempo
	created.Name = "João Silva"
	updated, err := auditRepo.Update(ctx, 1, created)
	assert.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt) || updated.UpdatedAt.Equal(updated.CreatedAt))
}