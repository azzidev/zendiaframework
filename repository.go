package zendia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository interface genérica para operações CRUD
type Repository[T any, ID comparable] interface {
	Create(ctx context.Context, entity T) (T, error)
	GetByID(ctx context.Context, id ID) (T, error)
	GetFirst(ctx context.Context, filters map[string]interface{}) (T, error)
	Update(ctx context.Context, id ID, entity T) (T, error)
	Delete(ctx context.Context, id ID) error
	GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error)
	GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error)
	List(ctx context.Context, filters map[string]interface{}) ([]T, error)
	Aggregate(ctx context.Context, pipeline []interface{}) ([]T, error)
}

// AuditInfo estrutura para informações de auditoria
type AuditInfo struct {
	SetAt  time.Time `bson:"set_at" json:"set_at"`
	ByName string    `bson:"by_name" json:"by_name"`
	ByID   uuid.UUID `bson:"by_id" json:"by_id"`
}

// AuditableEntity interface para entidades com auditoria
type AuditableEntity interface {
	SetCreated(AuditInfo)
	SetUpdated(AuditInfo)
	SetDeleted(AuditInfo)
	SetTenantID(string)
	SetActive(bool)
}

// AuditRepository wrapper que adiciona funcionalidades de auditoria
type AuditRepository[T any, ID comparable] struct {
	base Repository[T, ID]
}

// NewAuditRepository cria um repository com auditoria
func NewAuditRepository[T any, ID comparable](base Repository[T, ID]) *AuditRepository[T, ID] {
	return &AuditRepository[T, ID]{
		base: base,
	}
}

func (ar *AuditRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	tenantInfo := GetTenantInfo(ctx)
	
	if auditableEntity, ok := any(entity).(AuditableEntity); ok {
		var userID uuid.UUID
		if tenantInfo.UserID != "" {
			userID = uuid.MustParse(tenantInfo.UserID)
		}
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   userID,
		}
		auditableEntity.SetCreated(auditInfo)
		auditableEntity.SetUpdated(auditInfo)
		auditableEntity.SetActive(true)
		auditableEntity.SetTenantID(tenantInfo.TenantID)
	}
	
	return ar.base.Create(ctx, entity)
}

func (ar *AuditRepository[T, ID]) GetByID(ctx context.Context, id ID) (T, error) {
	// Para repositories não-MongoDB, não podemos injetar tenant automaticamente
	// pois não sabemos a estrutura dos filtros. Isso deve ser feito na camada de use case.
	return ar.base.GetByID(ctx, id)
}

func (ar *AuditRepository[T, ID]) Update(ctx context.Context, id ID, entity T) (T, error) {
	tenantInfo := GetTenantInfo(ctx)
	
	if auditableEntity, ok := any(entity).(AuditableEntity); ok {
		var userID uuid.UUID
		if tenantInfo.UserID != "" {
			userID = uuid.MustParse(tenantInfo.UserID)
		}
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   userID,
		}
		auditableEntity.SetUpdated(auditInfo)
		auditableEntity.SetTenantID(tenantInfo.TenantID)
	}
	
	return ar.base.Update(ctx, id, entity)
}

func (ar *AuditRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	return ar.base.Delete(ctx, id)
}

func (ar *AuditRepository[T, ID]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	// Injeta tenant_id e active automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if filters == nil {
		filters = make(map[string]interface{})
	}
	if tenantInfo.TenantID != "" {
		filters["tenant_id"] = tenantInfo.TenantID
	}
	filters["active"] = true
	return ar.base.GetFirst(ctx, filters)
}

func (ar *AuditRepository[T, ID]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// Injeta tenant_id e active automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if filters == nil {
		filters = make(map[string]interface{})
	}
	if tenantInfo.TenantID != "" {
		filters["tenant_id"] = tenantInfo.TenantID
	}
	filters["active"] = true
	return ar.base.GetAll(ctx, filters)
}

func (ar *AuditRepository[T, ID]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	// Injeta tenant_id e active automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if filters == nil {
		filters = make(map[string]interface{})
	}
	if tenantInfo.TenantID != "" {
		filters["tenant_id"] = tenantInfo.TenantID
	}
	filters["active"] = true
	return ar.base.GetAllSkipTake(ctx, filters, skip, take)
}

func (ar *AuditRepository[T, ID]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// Injeta tenant_id e active automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if filters == nil {
		filters = make(map[string]interface{})
	}
	if tenantInfo.TenantID != "" {
		filters["tenant_id"] = tenantInfo.TenantID
	}
	filters["active"] = true
	return ar.base.List(ctx, filters)
}

// MemoryRepository implementação em memória para testes
type MemoryRepository[T any, ID comparable] struct {
	data   map[ID]T
	nextID func() ID
}

// NewMemoryRepository cria um repository em memória
func NewMemoryRepository[T any, ID comparable](nextIDFunc func() ID) *MemoryRepository[T, ID] {
	return &MemoryRepository[T, ID]{
		data:   make(map[ID]T),
		nextID: nextIDFunc,
	}
}

func (mr *MemoryRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	id := mr.nextID()
	mr.data[id] = entity
	return entity, nil
}

func (mr *MemoryRepository[T, ID]) GetByID(ctx context.Context, id ID) (T, error) {
	entity, exists := mr.data[id]
	if !exists {
		var zero T
		return zero, NewNotFoundError("Entity not found")
	}
	return entity, nil
}

func (mr *MemoryRepository[T, ID]) Update(ctx context.Context, id ID, entity T) (T, error) {
	if _, exists := mr.data[id]; !exists {
		var zero T
		return zero, NewNotFoundError("Entity not found")
	}
	mr.data[id] = entity
	return entity, nil
}

func (mr *MemoryRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	if _, exists := mr.data[id]; !exists {
		return NewNotFoundError("Entity not found")
	}
	delete(mr.data, id)
	return nil
}

func (mr *MemoryRepository[T, ID]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	for _, entity := range mr.data {
		return entity, nil
	}
	var zero T
	return zero, NewNotFoundError("No entity found")
}

func (mr *MemoryRepository[T, ID]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	var result []T
	for _, entity := range mr.data {
		result = append(result, entity)
	}
	return result, nil
}

func (mr *MemoryRepository[T, ID]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	var result []T
	i := 0
	for _, entity := range mr.data {
		if i < skip {
			i++
			continue
		}
		if len(result) >= take {
			break
		}
		result = append(result, entity)
		i++
	}
	return result, nil
}

func (mr *MemoryRepository[T, ID]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	var result []T
	for _, entity := range mr.data {
		result = append(result, entity)
	}
	return result, nil
}

func (mr *MemoryRepository[T, ID]) Aggregate(ctx context.Context, pipeline []interface{}) ([]T, error) {
	return nil, NewInternalError("Aggregate not supported in memory repository")
}