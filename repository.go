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
	SetTenantID(string)
}

// LegacyAuditableEntity interface para compatibilidade com entidades antigas
type LegacyAuditableEntity interface {
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	SetCreatedBy(string)
	SetUpdatedBy(string)
	SetTenantID(string)
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
	
	// Tenta usar nova interface primeiro
	if newEntity, ok := any(entity).(AuditableEntity); ok {
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   uuid.MustParse(tenantInfo.UserID),
		}
		newEntity.SetCreated(auditInfo)
		newEntity.SetUpdated(auditInfo)
		newEntity.SetTenantID(tenantInfo.TenantID)
	} else if legacyEntity, ok := any(entity).(LegacyAuditableEntity); ok {
		// Fallback para interface antiga
		legacyEntity.SetCreatedAt(tenantInfo.ActionAt)
		legacyEntity.SetUpdatedAt(tenantInfo.ActionAt)
		legacyEntity.SetCreatedBy(tenantInfo.UserID)
		legacyEntity.SetUpdatedBy(tenantInfo.UserID)
		legacyEntity.SetTenantID(tenantInfo.TenantID)
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
	
	// Tenta usar nova interface primeiro
	if newEntity, ok := any(entity).(AuditableEntity); ok {
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   uuid.MustParse(tenantInfo.UserID),
		}
		newEntity.SetUpdated(auditInfo)
		newEntity.SetTenantID(tenantInfo.TenantID)
	} else if legacyEntity, ok := any(entity).(LegacyAuditableEntity); ok {
		// Fallback para interface antiga
		legacyEntity.SetUpdatedAt(tenantInfo.ActionAt)
		legacyEntity.SetUpdatedBy(tenantInfo.UserID)
		legacyEntity.SetTenantID(tenantInfo.TenantID)
	}
	
	return ar.base.Update(ctx, id, entity)
}

func (ar *AuditRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	return ar.base.Delete(ctx, id)
}

func (ar *AuditRepository[T, ID]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	// Injeta tenant_id automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" && filters != nil {
		filters["tenant_id"] = tenantInfo.TenantID
	}
	return ar.base.GetFirst(ctx, filters)
}

func (ar *AuditRepository[T, ID]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// Injeta tenant_id automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantInfo.TenantID
	}
	return ar.base.GetAll(ctx, filters)
}

func (ar *AuditRepository[T, ID]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	// Injeta tenant_id automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantInfo.TenantID
	}
	return ar.base.GetAllSkipTake(ctx, filters, skip, take)
}

func (ar *AuditRepository[T, ID]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// Injeta tenant_id automaticamente nos filtros
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantInfo.TenantID
	}
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