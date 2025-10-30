package zendia

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository implementação do Repository para MongoDB
type MongoRepository[T any, ID comparable] struct {
	collection *mongo.Collection
	idField    string
}

// allowedFilterKeys defines safe field names for filtering based on existing AuditInfo structure
var allowedFilterKeys = map[string]bool{
	"_id":             true,
	"tenant_id":       true,
	"name":            true,
	"email":           true,
	"status":          true,
	"active":          true,
	"created.set_at":  true,
	"created.by_name": true,
	"created.by_id":   true,
	"created.active":  true,
	"updated.set_at":  true,
	"updated.by_name": true,
	"updated.by_id":   true,
	"deleted.set_at":  true,
	"deleted.by_name": true,
	"deleted.by_id":   true,
	"deleted.active":  true,
}

// sanitizeFilters prevents NoSQL injection by validating and sanitizing filters
func sanitizeFilters(filters map[string]interface{}) (bson.M, error) {
	if len(filters) > 20 { // Prevent DoS with too many filters
		return nil, fmt.Errorf("too many filters provided")
	}

	sanitized := bson.M{}
	for key, value := range filters {
		// Validate field name
		if !isValidFieldName(key) {
			log.Printf("Invalid field name rejected: %s", key)
			continue // Skip invalid field names
		}

		// Sanitize value based on type
		sanitizedValue, err := sanitizeFilterValue(value)
		if err != nil {
			log.Printf("Invalid filter value for field %s: %v", key, err)
			continue // Skip invalid values
		}

		sanitized[key] = sanitizedValue
	}

	return sanitized, nil
}

// isValidFieldName checks if field name is safe for MongoDB queries
func isValidFieldName(fieldName string) bool {
	// Check against whitelist
	if allowedFilterKeys[fieldName] {
		return true
	}

	// Allow custom fields that match safe pattern
	validPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,50}$`)
	if !validPattern.MatchString(fieldName) {
		return false
	}

	// Reject MongoDB operators and dangerous patterns
	dangerousPatterns := []string{"$", ".", "javascript", "eval", "function", "where"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToLower(fieldName), pattern) {
			return false
		}
	}

	return true
}

// sanitizeFilterValue sanitizes filter values to prevent injection
func sanitizeFilterValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		// Limit string length and check for dangerous patterns
		if len(v) > 1000 {
			return nil, fmt.Errorf("string value too long")
		}
		// Check for MongoDB operators and JavaScript
		dangerousPatterns := []string{"$", "javascript:", "eval(", "function(", "where"}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(v), pattern) {
				return nil, fmt.Errorf("dangerous pattern detected")
			}
		}
		return v, nil
	case int, int32, int64, float32, float64, bool:
		return v, nil
	case primitive.ObjectID:
		return v, nil
	case uuid.UUID:
		return primitive.Binary{Subtype: 4, Data: v[:]}, nil
	case map[string]interface{}:
		// Recursively sanitize nested objects (limited depth)
		return sanitizeNestedObject(v, 1)
	default:
		// Use reflection for other types but be restrictive
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			if val.Len() > 100 { // Prevent DoS with large arrays
				return nil, fmt.Errorf("array too large")
			}
		}
		return v, nil
	}
}

// sanitizeNestedObject sanitizes nested objects with depth limit
func sanitizeNestedObject(obj map[string]interface{}, depth int) (map[string]interface{}, error) {
	if depth > 3 { // Prevent deep nesting attacks
		return nil, fmt.Errorf("object nesting too deep")
	}

	sanitized := make(map[string]interface{})
	for key, value := range obj {
		if !isValidFieldName(key) {
			continue
		}
		sanitizedValue, err := sanitizeFilterValue(value)
		if err != nil {
			continue
		}
		sanitized[key] = sanitizedValue
	}

	return sanitized, nil
}

// NewMongoRepository creates a new MongoDB repository with security validations
func NewMongoRepository[T any, ID comparable](collection *mongo.Collection, idField string) *MongoRepository[T, ID] {
	if idField == "" {
		idField = "_id"
	}
	// Validate idField to prevent injection
	if !isValidFieldName(idField) {
		log.Printf("Warning: potentially unsafe idField: %s", idField)
		idField = "_id" // Fallback to safe default
	}
	return &MongoRepository[T, ID]{
		collection: collection,
		idField:    idField,
	}
}

func (mr *MongoRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	_, err := mr.collection.InsertOne(ctx, entity)
	if err != nil {
		var zero T
		return zero, NewInternalError("Failed to create entity: " + err.Error())
	}

	return entity, nil
}

func (mr *MongoRepository[T, ID]) GetByID(ctx context.Context, id ID) (T, error) {
	var entity T
	filter := bson.M{mr.idField: id}

	err := mr.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("Entity not found")
		}
		return entity, NewInternalError("Failed to get entity: " + err.Error())
	}

	return entity, nil
}

func (mr *MongoRepository[T, ID]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	var entity T

	// Sanitize filters to prevent NoSQL injection
	filter, err := sanitizeFilters(filters)
	if err != nil {
		log.Printf("Filter sanitization failed: %v", err)
		return entity, NewBadRequestError("Invalid filter parameters")
	}

	err = mr.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("No entity found")
		}
		return entity, NewInternalError("Failed to get first entity: " + err.Error())
	}

	return entity, nil
}

func (mr *MongoRepository[T, ID]) Update(ctx context.Context, id ID, entity T) (T, error) {
	filter := bson.M{mr.idField: id}
	update := bson.M{"$set": entity}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated T

	err := mr.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return updated, NewNotFoundError("Entity not found")
		}
		return updated, NewInternalError("Failed to update entity: " + err.Error())
	}

	return updated, nil
}

func (mr *MongoRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	filter := bson.M{mr.idField: id}

	result, err := mr.collection.DeleteOne(ctx, filter)
	if err != nil {
		return NewInternalError("Failed to delete entity: " + err.Error())
	}

	if result.DeletedCount == 0 {
		return NewNotFoundError("Entity not found")
	}

	return nil
}

func (mr *MongoRepository[T, ID]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	// Sanitize filters to prevent NoSQL injection
	filter, err := sanitizeFilters(filters)
	if err != nil {
		log.Printf("Filter sanitization failed: %v", err)
		return nil, NewBadRequestError("Invalid filter parameters")
	}

	cursor, err := mr.collection.Find(ctx, filter)
	if err != nil {
		return nil, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, nil
}

func (mr *MongoRepository[T, ID]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	// Validate pagination parameters
	if skip < 0 || take < 0 || take > 1000 {
		return nil, NewBadRequestError("Invalid pagination parameters")
	}

	// Sanitize filters to prevent NoSQL injection
	filter, err := sanitizeFilters(filters)
	if err != nil {
		log.Printf("Filter sanitization failed: %v", err)
		return nil, NewBadRequestError("Invalid filter parameters")
	}

	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(take))

	cursor, err := mr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, nil
}

func (mr *MongoRepository[T, ID]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	return mr.GetAll(ctx, filters)
}

// MongoAuditableEntity interface para entidades MongoDB com auditoria
type MongoAuditableEntity interface {
	GetID() uuid.UUID
	SetID(uuid.UUID)
	SetTenantID(string)
}

// MongoAuditRepository repository MongoDB com auditoria
type MongoAuditRepository[T MongoAuditableEntity] struct {
	base *MongoRepository[T, uuid.UUID]
}

// NewMongoAuditRepository cria um repository MongoDB com auditoria
func NewMongoAuditRepository[T MongoAuditableEntity](collection *mongo.Collection) *MongoAuditRepository[T] {
	base := NewMongoRepository[T, uuid.UUID](collection, "_id")
	return &MongoAuditRepository[T]{
		base: base,
	}
}

func (mar *MongoAuditRepository[T]) Create(ctx context.Context, entity T) (T, error) {
	tenantInfo := GetTenantInfo(ctx)

	// Gera UUID se não tiver ID
	if entity.GetID() == uuid.Nil {
		entity.SetID(uuid.New())
	}

	// Tenta usar nova interface primeiro
	if newEntity, ok := any(entity).(AuditableEntity); ok {
		var userID uuid.UUID
		if tenantInfo.UserID != "" {
			userID = uuid.MustParse(tenantInfo.UserID)
		}
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   userID,
		}
		newEntity.SetCreated(auditInfo)
		newEntity.SetUpdated(auditInfo)
	} else if legacyEntity, ok := any(entity).(LegacyAuditableEntity); ok {
		// Fallback para interface antiga
		legacyEntity.SetCreatedAt(tenantInfo.ActionAt)
		legacyEntity.SetUpdatedAt(tenantInfo.ActionAt)
		legacyEntity.SetCreatedBy(tenantInfo.UserID)
		legacyEntity.SetUpdatedBy(tenantInfo.UserID)
	}

	entity.SetTenantID(tenantInfo.TenantID)

	// Converte UUIDs para binary subtype 4
	doc := convertUUIDs(entity)
	_, err := mar.base.collection.InsertOne(ctx, doc)
	if err != nil {
		var zero T
		return zero, NewInternalError("Failed to create entity: " + err.Error())
	}

	return entity, nil
}

func (mar *MongoAuditRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	var entity T
	binaryUUID := primitive.Binary{Subtype: 4, Data: id[:]}
	filter := bson.M{
		"_id":     binaryUUID,
		"deleted": nil,
	}

	// Injeta tenant_id automaticamente
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	err := mar.base.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("Entity not found")
		}
		return entity, NewInternalError("Failed to get entity: " + err.Error())
	}

	return entity, nil
}

func (mar *MongoAuditRepository[T]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	var entity T
	filter := bson.M{
		"deleted": nil,
	}

	// Inject tenant_id automatically for security
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		} else {
			log.Printf("Invalid tenant ID format: %s", tenantInfo.TenantID)
			return entity, NewBadRequestError("Invalid tenant ID")
		}
	}

	// Sanitize user filters to prevent NoSQL injection
	sanitizedFilters, err := sanitizeFilters(filters)
	if err != nil {
		log.Printf("Filter sanitization failed: %v", err)
		return entity, NewBadRequestError("Invalid filter parameters")
	}

	// Merge sanitized filters
	for k, v := range sanitizedFilters {
		filter[k] = v
	}

	err = mar.base.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("No entity found")
		}
		return entity, NewInternalError("Failed to get first entity: " + err.Error())
	}

	return entity, nil
}

func (mar *MongoAuditRepository[T]) Update(ctx context.Context, id uuid.UUID, entity T) (T, error) {
	tenantInfo := GetTenantInfo(ctx)

	if auditEntity, ok := any(entity).(AuditableEntity); ok {
		var userID uuid.UUID
		if tenantInfo.UserID != "" {
			userID = uuid.MustParse(tenantInfo.UserID)
		}
		auditInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   userID,
		}
		auditEntity.SetUpdated(auditInfo)
	}

	entity.SetTenantID(tenantInfo.TenantID)

	binaryUUID := primitive.Binary{Subtype: 4, Data: id[:]}
	filter := bson.M{"_id": binaryUUID}

	// Injeta tenant_id automaticamente no filtro de update
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}
	doc := convertUUIDs(entity)
	update := bson.M{"$set": doc}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated T

	err := mar.base.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return updated, NewNotFoundError("Entity not found")
		}
		return updated, NewInternalError("Failed to update entity: " + err.Error())
	}

	return updated, nil
}

func (mar *MongoAuditRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	tenantInfo := GetTenantInfo(ctx)

	// Busca a entidade primeiro
	entity, err := mar.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Seta deleted info se a entidade suporta
	if auditEntity, ok := any(entity).(AuditableEntity); ok {
		var userID uuid.UUID
		if tenantInfo.UserID != "" {
			userID = uuid.MustParse(tenantInfo.UserID)
		}
		deleteInfo := AuditInfo{
			SetAt:  tenantInfo.ActionAt,
			ByName: tenantInfo.UserName,
			ByID:   userID,
			Active: false,
		}
		auditEntity.SetDeleted(deleteInfo)

		// Atualiza a entidade
		_, err = mar.Update(ctx, id, entity)
		return err
	}

	// Fallback para entidades antigas
	binaryUUID := primitive.Binary{Subtype: 4, Data: id[:]}
	filter := bson.M{"_id": binaryUUID}
	if tenantInfo.TenantID != "" {
		tenantUUID, _ := uuid.Parse(tenantInfo.TenantID)
		filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": tenantInfo.ActionAt,
			"deleted_by": tenantInfo.UserID,
		},
	}

	result, err := mar.base.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return NewInternalError("Failed to soft delete entity: " + err.Error())
	}

	if result.ModifiedCount == 0 {
		return NewNotFoundError("Entity not found or already deleted")
	}

	return nil
}

func (mar *MongoAuditRepository[T]) GetAll(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	filter := bson.M{
		"deleted": nil,
	}

	// Inject tenant_id automatically for security
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		} else {
			log.Printf("Invalid tenant ID format: %s", tenantInfo.TenantID)
			return nil, NewBadRequestError("Invalid tenant ID")
		}
	}

	// Sanitize user filters to prevent NoSQL injection
	sanitizedFilters, err := sanitizeFilters(filters)
	if err != nil {
		log.Printf("Filter sanitization failed: %v", err)
		return nil, NewBadRequestError("Invalid filter parameters")
	}

	// Merge sanitized filters
	for k, v := range sanitizedFilters {
		filter[k] = v
	}

	cursor, err := mar.base.collection.Find(ctx, filter)
	if err != nil {
		return nil, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, nil
}

func (mar *MongoAuditRepository[T]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int) ([]T, error) {
	filter := bson.M{
		"deleted": nil,
	}

	// Injeta tenant_id automaticamente
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
	}

	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(take))

	cursor, err := mar.base.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, nil
}

func (mar *MongoAuditRepository[T]) List(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	return mar.GetAll(ctx, filters)
}

// GetAllIncludingDeleted busca todos os registros incluindo os deletados
func (mar *MongoAuditRepository[T]) GetAllIncludingDeleted(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	filter := bson.M{}

	// Injeta tenant_id automaticamente
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
	}

	cursor, err := mar.base.collection.Find(ctx, filter)
	if err != nil {
		return nil, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, nil
}

// GetDeleted busca apenas registros deletados
func (mar *MongoAuditRepository[T]) GetDeleted(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	filter := bson.M{
		"deleted": bson.M{"$ne": nil}, // Apenas registros deletados
	}

	// Injeta tenant_id automaticamente
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
	}

	cursor, err := mar.base.collection.Find(ctx, filter)
	if err != nil {
		return nil, NewInternalError("Failed to get deleted entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, NewInternalError("Failed to decode deleted entities: " + err.Error())
	}

	return entities, nil
}

// HardDelete remove permanentemente do banco
func (mar *MongoAuditRepository[T]) HardDelete(ctx context.Context, id uuid.UUID) error {
	binaryUUID := primitive.Binary{Subtype: 4, Data: id[:]}
	filter := bson.M{"_id": binaryUUID}

	// Injeta tenant_id automaticamente no filtro de hard delete
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	result, err := mar.base.collection.DeleteOne(ctx, filter)
	if err != nil {
		return NewInternalError("Failed to hard delete entity: " + err.Error())
	}

	if result.DeletedCount == 0 {
		return NewNotFoundError("Entity not found")
	}

	return nil
}

// Restore restaura um registro soft deleted
func (mar *MongoAuditRepository[T]) Restore(ctx context.Context, id uuid.UUID) error {
	binaryUUID := primitive.Binary{Subtype: 4, Data: id[:]}
	filter := bson.M{
		"_id":     binaryUUID,
		"deleted": bson.M{"$ne": nil}, // Só restaura se estiver deletado
	}

	// Injeta tenant_id automaticamente no filtro de restore
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = primitive.Binary{Subtype: 4, Data: tenantUUID[:]}
		}
	}

	update := bson.M{
		"$unset": bson.M{
			"deleted": "",
		},
	}

	result, err := mar.base.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return NewInternalError("Failed to restore entity: " + err.Error())
	}

	if result.ModifiedCount == 0 {
		return NewNotFoundError("Entity not found or not deleted")
	}

	return nil
}
