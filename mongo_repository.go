package zendia

import (
	"context"

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

// NewMongoRepository cria um novo repository MongoDB
func NewMongoRepository[T any, ID comparable](collection *mongo.Collection, idField string) *MongoRepository[T, ID] {
	if idField == "" {
		idField = "_id"
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
	filter := bson.M{}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
	}

	err := mr.collection.FindOne(ctx, filter).Decode(&entity)
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
	filter := bson.M{}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
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
	filter := bson.M{}

	// Converte filtros para BSON
	for k, v := range filters {
		filter[k] = v
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

	err := mar.base.collection.FindOne(ctx, filter).Decode(&entity)
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
