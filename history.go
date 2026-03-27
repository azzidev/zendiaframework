package zendia

import (
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

// TriggerInfo informações sobre o trigger que causou a mudança
type TriggerInfo struct {
	Name string    `bson:"name" json:"name"`
	At   time.Time `bson:"at" json:"at"`
	By   string    `bson:"by" json:"by"`
}

// HistoryEntry representa uma entrada no histórico de mudanças
type HistoryEntry struct {
	ID         uuid.UUID              `bson:"_id" json:"id"`
	EntityID   uuid.UUID              `bson:"entity_id" json:"entityId"`
	EntityType string                 `bson:"entity_type" json:"entityType"`
	TenantID   uuid.UUID              `bson:"tenant_id" json:"tenantId"`
	Trigger    TriggerInfo            `bson:"trigger" json:"trigger"`
	Changes    map[string]FieldChange `bson:"changes" json:"changes"`
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
	return &HistoryManager{collection: collection}
}

// RecordChanges registra as mudanças entre dois objetos
func (hm *HistoryManager) RecordChanges(ctx context.Context, entityID uuid.UUID, entityType, triggerName string, before, after interface{}) error {
	tenantInfo := GetTenantInfo(ctx)

	changes := hm.detectChanges(before, after)
	if len(changes) == 0 {
		return nil
	}

	var tenantUUID uuid.UUID
	if tenantInfo.TenantID != "" {
		tenantUUID = uuid.MustParse(tenantInfo.TenantID)
	}

	entry := HistoryEntry{
		ID:         uuid.New(),
		EntityID:   entityID,
		EntityType: entityType,
		TenantID:   tenantUUID,
		Trigger: TriggerInfo{
			Name: triggerName,
			At:   tenantInfo.ActionAt,
			By:   tenantInfo.UserName,
		},
		Changes: changes,
	}

	_, err := hm.collection.InsertOne(ctx, entry)
	return err
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

func (hm *HistoryManager) detectChanges(before, after interface{}) map[string]FieldChange {
	changes := make(map[string]FieldChange)

	beforeVal := reflect.ValueOf(before)
	afterVal := reflect.ValueOf(after)

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

		if hm.shouldSkipField(fieldName) {
			continue
		}

		beforeFieldVal := beforeVal.Field(i)
		afterFieldVal := afterVal.Field(i)

		if !reflect.DeepEqual(beforeFieldVal.Interface(), afterFieldVal.Interface()) {
			changes[fieldName] = FieldChange{
				Before: beforeFieldVal.Interface(),
				After:  afterFieldVal.Interface(),
			}
		}
	}

	return changes
}

func (hm *HistoryManager) shouldSkipField(fieldName string) bool {
	skipFields := map[string]bool{
		"Created":   true,
		"Updated":   true,
		"Deleted":   true,
		"DeletedAt": true,
		"DeletedBy": true,
		"CreatedAt": true,
		"UpdatedAt": true,
		"CreatedBy": true,
		"UpdatedBy": true,
		"TenantID":  true,
		"ID":        true,
	}
	return skipFields[fieldName]
}
