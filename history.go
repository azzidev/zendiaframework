package zendia

import (
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

// HistoryEntry representa uma entrada no histórico de mudanças
type HistoryEntry struct {
	ID          uuid.UUID              `bson:"_id" json:"id"`
	EntityID    uuid.UUID              `bson:"entity_id" json:"entityId"`
	EntityType  string                 `bson:"entity_type" json:"entityType"`
	TenantID    uuid.UUID              `bson:"tenant_id" json:"tenantId"`
	TriggerName string                 `bson:"trigger_name" json:"triggerName"`
	TriggerAt   time.Time              `bson:"trigger_at" json:"triggerAt"`
	TriggerBy   string                 `bson:"trigger_by" json:"triggerBy"`
	Changes     map[string]FieldChange `bson:"changes" json:"changes"`
}

// FieldChange representa a mudança de um campo específico
type FieldChange struct {
	Before interface{} `bson:"before" json:"before"`
	After  interface{} `bson:"after" json:"after"`
}

// HistoryManager gerencia o histórico de mudanças
type HistoryManager struct {
	collection *mongo.Collection
}

// NewHistoryManager cria um novo gerenciador de histórico
func NewHistoryManager(collection *mongo.Collection) *HistoryManager {
	return &HistoryManager{
		collection: collection,
	}
}

// RecordChanges registra as mudanças entre dois objetos
func (hm *HistoryManager) RecordChanges(ctx context.Context, entityID uuid.UUID, entityType, triggerName string, before, after interface{}) error {
	tenantInfo := GetTenantInfo(ctx)

	changes := hm.detectChanges(before, after)
	if len(changes) == 0 {
		return nil // Nenhuma mudança detectada
	}

	var tenantUUID uuid.UUID
	if tenantInfo.TenantID != "" {
		tenantUUID = uuid.MustParse(tenantInfo.TenantID)
	}

	entry := HistoryEntry{
		ID:          uuid.New(),
		EntityID:    entityID,
		EntityType:  entityType,
		TenantID:    tenantUUID,
		TriggerName: triggerName,
		TriggerAt:   tenantInfo.ActionAt,
		TriggerBy:   tenantInfo.UserName,
		Changes:     changes,
	}

	_, err := hm.collection.InsertOne(ctx, entry)
	return err
}

// detectChanges compara dois objetos e retorna apenas os campos que mudaram
func (hm *HistoryManager) detectChanges(before, after interface{}) map[string]FieldChange {
	changes := make(map[string]FieldChange)

	beforeVal := reflect.ValueOf(before)
	afterVal := reflect.ValueOf(after)

	// Se são ponteiros, pega o valor
	if beforeVal.Kind() == reflect.Ptr {
		beforeVal = beforeVal.Elem()
	}
	if afterVal.Kind() == reflect.Ptr {
		afterVal = afterVal.Elem()
	}

	beforeType := beforeVal.Type()

	for i := 0; i < beforeVal.NumField(); i++ {
		field := beforeType.Field(i)
		fieldName := field.Name

		// Pula campos de auditoria e sistema
		if hm.shouldSkipField(fieldName) {
			continue
		}

		beforeFieldVal := beforeVal.Field(i)
		afterFieldVal := afterVal.Field(i)

		// Compara os valores
		if !reflect.DeepEqual(beforeFieldVal.Interface(), afterFieldVal.Interface()) {
			changes[fieldName] = FieldChange{
				Before: beforeFieldVal.Interface(),
				After:  afterFieldVal.Interface(),
			}
		}
	}

	return changes
}

// shouldSkipField verifica se um campo deve ser ignorado no histórico
func (hm *HistoryManager) shouldSkipField(fieldName string) bool {
	skipFields := []string{
		"Created", "Updated", "DeletedAt", "DeletedBy",
		"CreatedAt", "UpdatedAt", "CreatedBy", "UpdatedBy",
		"TenantID", "ID",
	}

	for _, skip := range skipFields {
		if fieldName == skip {
			return true
		}
	}
	return false
}

// GetHistory busca o histórico de uma entidade
func (hm *HistoryManager) GetHistory(ctx context.Context, entityID uuid.UUID) ([]HistoryEntry, error) {
	tenantInfo := GetTenantInfo(ctx)

	filter := map[string]interface{}{
		"entity_id": entityID,
	}
	
	if tenantInfo.TenantID != "" {
		filter["tenant_id"] = uuid.MustParse(tenantInfo.TenantID)
	}

	cursor, err := hm.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var history []HistoryEntry
	if err = cursor.All(ctx, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// HistoryAuditRepository repository com histórico automático
type HistoryAuditRepository[T MongoAuditableEntity] struct {
	base       *MongoAuditRepository[T]
	history    *HistoryManager
	entityType string
}

// NewHistoryAuditRepository cria repository com histórico
func NewHistoryAuditRepository[T MongoAuditableEntity](collection *mongo.Collection, historyCollection *mongo.Collection, entityType string) *HistoryAuditRepository[T] {
	base := NewMongoAuditRepository[T](collection)
	history := NewHistoryManager(historyCollection)

	return &HistoryAuditRepository[T]{
		base:       base,
		history:    history,
		entityType: entityType,
	}
}

func (har *HistoryAuditRepository[T]) Create(ctx context.Context, entity T) (T, error) {
	return har.base.Create(ctx, entity)
}

func (har *HistoryAuditRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	return har.base.GetByID(ctx, id)
}

func (har *HistoryAuditRepository[T]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	return har.base.GetFirst(ctx, filters)
}

func (har *HistoryAuditRepository[T]) Update(ctx context.Context, id uuid.UUID, entity T) (T, error) {
	// Busca o estado anterior
	before, err := har.base.GetByID(ctx, id)
	if err != nil {
		return entity, err
	}

	// Atualiza
	updated, err := har.base.Update(ctx, id, entity)
	if err != nil {
		return entity, err
	}

	// Registra histórico
	har.history.RecordChanges(ctx, id, har.entityType, "Update", before, updated)

	return updated, nil
}

func (har *HistoryAuditRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	return har.base.Delete(ctx, id)
}

func (har *HistoryAuditRepository[T]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	return har.base.GetAll(ctx, filters)
}

func (har *HistoryAuditRepository[T]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	return har.base.GetAllSkipTake(ctx, filters, skip, take)
}

func (har *HistoryAuditRepository[T]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	return har.base.List(ctx, filters)
}

// GetHistory busca histórico da entidade
func (har *HistoryAuditRepository[T]) GetHistory(ctx context.Context, entityID uuid.UUID) ([]HistoryEntry, error) {
	return har.history.GetHistory(ctx, entityID)
}
