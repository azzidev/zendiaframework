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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RepositoryConfig configuração do repository
type RepositoryConfig struct {
	audit      bool
	history    bool
	historyCol *mongo.Collection
	entityType string
}

// RepositoryOption função para configurar o repository
type RepositoryOption func(*RepositoryConfig)

// WithAudit habilita auditoria automática (created, updated, deleted, tenant)
func WithAudit() RepositoryOption {
	return func(c *RepositoryConfig) {
		c.audit = true
	}
}

// WithHistory habilita tracking de histórico de mudanças
func WithHistory(historyCollection *mongo.Collection, entityType string) RepositoryOption {
	return func(c *RepositoryConfig) {
		c.history = true
		c.historyCol = historyCollection
		c.entityType = entityType
	}
}

// Repository implementação unificada para MongoDB
type Repository[T MongoAuditableEntity] struct {
	collection *mongo.Collection
	config     RepositoryConfig
	history    *HistoryManager
}

// NewRepository cria um novo repository MongoDB
//
// Uso:
//
//	repo := zendia.NewRepository[*User](collection)
//	repo := zendia.NewRepository[*User](collection, zendia.WithAudit())
//	repo := zendia.NewRepository[*User](collection, zendia.WithAudit(), zendia.WithHistory(historyCol, "User"))
func NewRepository[T MongoAuditableEntity](collection *mongo.Collection, opts ...RepositoryOption) *Repository[T] {
	cfg := RepositoryConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var hm *HistoryManager
	if cfg.history && cfg.historyCol != nil {
		hm = NewHistoryManager(cfg.historyCol)
	}

	return &Repository[T]{
		collection: collection,
		config:     cfg,
		history:    hm,
	}
}

func (r *Repository[T]) Create(ctx context.Context, entity T) (T, error) {
	if entity.GetID() == uuid.Nil {
		entity.SetID(uuid.New())
	}

	if r.config.audit {
		tenantInfo := GetTenantInfo(ctx)
		entity.SetTenantID(tenantInfo.TenantID)

		if ae, ok := any(entity).(AuditableEntity); ok {
			info := r.buildAuditInfo(tenantInfo)
			ae.SetCreated(info)
			ae.SetUpdated(info)
			ae.SetActive(true)
		}
	}

	_, err := r.collection.InsertOne(ctx, entity)
	if err != nil {
		var zero T
		return zero, NewInternalError("Failed to create entity: " + err.Error())
	}

	return entity, nil
}

func (r *Repository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	var entity T
	filter := bson.M{
		"_id":    id,
		"active": true,
	}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	err := r.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("Entity not found")
		}
		return entity, NewInternalError("Failed to get entity: " + err.Error())
	}

	return entity, nil
}

func (r *Repository[T]) GetFirst(ctx context.Context, filters map[string]interface{}) (T, error) {
	var entity T
	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	err := r.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity, NewNotFoundError("No entity found")
		}
		return entity, NewInternalError("Failed to get first entity: " + err.Error())
	}

	return entity, nil
}

func (r *Repository[T]) Update(ctx context.Context, id uuid.UUID, entity T) (T, error) {
	// Se history está habilitado, busca o estado anterior
	var before T
	if r.config.history && r.history != nil {
		var err error
		before, err = r.GetByID(ctx, id)
		if err != nil {
			return entity, err
		}
	}

	if r.config.audit {
		tenantInfo := GetTenantInfo(ctx)
		entity.SetTenantID(tenantInfo.TenantID)

		if ae, ok := any(entity).(AuditableEntity); ok {
			ae.SetUpdated(r.buildAuditInfo(tenantInfo))
		}
	}

	filter := bson.M{"_id": id}
	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	update := bson.M{"$set": entity}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated T

	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return updated, NewNotFoundError("Entity not found")
		}
		return updated, NewInternalError("Failed to update entity: " + err.Error())
	}

	// Registra histórico se habilitado
	if r.config.history && r.history != nil {
		r.history.RecordChanges(ctx, id, r.config.entityType, "Update", before, updated)
	}

	return updated, nil
}

func (r *Repository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	if r.config.audit {
		entity, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if ae, ok := any(entity).(AuditableEntity); ok {
			tenantInfo := GetTenantInfo(ctx)
			ae.SetDeleted(r.buildAuditInfo(tenantInfo))
			ae.SetActive(false)
			_, err = r.Update(ctx, id, entity)
			return err
		}
	}

	// Sem audit: soft delete simples
	filter := bson.M{"_id": id, "active": true}
	update := bson.M{"$set": bson.M{"active": false}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return NewInternalError("Failed to delete entity: " + err.Error())
	}
	if result.ModifiedCount == 0 {
		return NewNotFoundError("Entity not found or already deleted")
	}

	return nil
}

func (r *Repository[T]) GetAll(ctx context.Context, filters map[string]interface{}, opts ...*QueryOptions) ([]T, error) {
	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	findOpts := options.Find()
	r.applyQueryOptions(findOpts, opts...)

	cursor, err := r.collection.Find(ctx, filter, findOpts)
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

func (r *Repository[T]) GetAllSkipTake(ctx context.Context, filters map[string]interface{}, skip, take int, opts ...*QueryOptions) ([]T, error) {
	if skip < 0 || take < 0 || take > 1000 {
		return nil, NewBadRequestError("Invalid pagination parameters")
	}

	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	findOpts := options.Find().SetSkip(int64(skip)).SetLimit(int64(take))
	r.applyQueryOptions(findOpts, opts...)

	cursor, err := r.collection.Find(ctx, filter, findOpts)
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

func (r *Repository[T]) List(ctx context.Context, filters map[string]interface{}, opts ...*QueryOptions) ([]T, error) {
	return r.GetAll(ctx, filters, opts...)
}

// Count retorna o total de documentos que correspondem aos filtros
func (r *Repository[T]) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, NewInternalError("Failed to count entities: " + err.Error())
	}

	return count, nil
}

// CountAll retorna o total incluindo deletados
func (r *Repository[T]) CountAll(ctx context.Context, filters map[string]interface{}) (int64, error) {
	filter := bson.M{}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, NewInternalError("Failed to count entities: " + err.Error())
	}

	return count, nil
}

// GetAllSkipTakeWithCount retorna os documentos paginados + total em uma única chamada
func (r *Repository[T]) GetAllSkipTakeWithCount(ctx context.Context, filters map[string]interface{}, skip, take int, opts ...*QueryOptions) ([]T, int64, error) {
	if skip < 0 || take < 0 || take > 1000 {
		return nil, 0, NewBadRequestError("Invalid pagination parameters")
	}

	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, NewInternalError("Failed to count entities: " + err.Error())
	}

	findOpts := options.Find().SetSkip(int64(skip)).SetLimit(int64(take))
	r.applyQueryOptions(findOpts, opts...)

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, NewInternalError("Failed to get entities: " + err.Error())
	}
	defer cursor.Close(ctx)

	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return nil, 0, NewInternalError("Failed to decode entities: " + err.Error())
	}

	return entities, count, nil
}

func (r *Repository[T]) Aggregate(ctx context.Context, pipeline []interface{}) ([]T, error) {
	matchFilter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, matchFilter)
	}

	auditFilter := bson.M{"$match": matchFilter}
	fullPipeline := append([]interface{}{auditFilter}, pipeline...)

	cursor, err := r.collection.Aggregate(ctx, fullPipeline)
	if err != nil {
		return nil, NewInternalError("Failed to aggregate: " + err.Error())
	}
	defer cursor.Close(ctx)

	var results []T
	if err = cursor.All(ctx, &results); err != nil {
		return nil, NewInternalError("Failed to decode aggregate results: " + err.Error())
	}

	return results, nil
}

func (r *Repository[T]) AggregateRaw(ctx context.Context, pipeline []interface{}) ([]map[string]interface{}, error) {
	matchFilter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, matchFilter)
	}

	auditFilter := bson.M{"$match": matchFilter}
	fullPipeline := append([]interface{}{auditFilter}, pipeline...)

	cursor, err := r.collection.Aggregate(ctx, fullPipeline)
	if err != nil {
		return nil, NewInternalError("Failed to aggregate: " + err.Error())
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err = cursor.All(ctx, &results); err != nil {
		return nil, NewInternalError("Failed to decode aggregate results: " + err.Error())
	}

	return results, nil
}

func (r *Repository[T]) GetHistory(ctx context.Context, entityID uuid.UUID) ([]HistoryEntry, error) {
	if r.history == nil {
		return nil, NewBadRequestError("History not enabled for this repository")
	}
	return r.history.GetHistory(ctx, entityID)
}

// GetAllIncludingDeleted busca todos os registros incluindo os deletados
func (r *Repository[T]) GetAllIncludingDeleted(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	filter := bson.M{}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	cursor, err := r.collection.Find(ctx, filter)
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

// GetDeleted busca apenas registros deletados (active=false)
func (r *Repository[T]) GetDeleted(ctx context.Context, filters map[string]interface{}) ([]T, error) {
	filter := bson.M{"active": false}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	cursor, err := r.collection.Find(ctx, filter)
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
func (r *Repository[T]) HardDelete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return NewInternalError("Failed to hard delete entity: " + err.Error())
	}
	if result.DeletedCount == 0 {
		return NewNotFoundError("Entity not found")
	}

	return nil
}

// Restore restaura um registro soft deleted
func (r *Repository[T]) Restore(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{
		"_id":    id,
		"active": false,
	}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	update := bson.M{
		"$set": bson.M{"active": true},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return NewInternalError("Failed to restore entity: " + err.Error())
	}
	if result.ModifiedCount == 0 {
		return NewNotFoundError("Entity not found or not deleted")
	}

	return nil
}

// DeleteMany soft delete múltiplos registros
func (r *Repository[T]) DeleteMany(ctx context.Context, filters map[string]interface{}) (int64, error) {
	filter := bson.M{"active": true}

	if r.config.audit {
		r.injectTenantFilter(ctx, filter)
	}

	for k, v := range filters {
		filter[k] = v
	}

	updateFields := bson.M{"active": false}

	if r.config.audit {
		tenantInfo := GetTenantInfo(ctx)
		updateFields["deleted"] = r.buildAuditInfo(tenantInfo)
	}

	update := bson.M{"$set": updateFields}

	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, NewInternalError("Failed to delete entities: " + err.Error())
	}

	return result.ModifiedCount, nil
}

// --- helpers ---

func (r *Repository[T]) buildAuditInfo(tenantInfo TenantInfo) AuditInfo {
	var userID uuid.UUID
	if tenantInfo.UserID != "" {
		userID = uuid.MustParse(tenantInfo.UserID)
	}
	return AuditInfo{
		SetAt:  tenantInfo.ActionAt,
		ByName: tenantInfo.UserName,
		ByID:   userID,
	}
}

func (r *Repository[T]) injectTenantFilter(ctx context.Context, filter bson.M) {
	tenantInfo := GetTenantInfo(ctx)
	if tenantInfo.TenantID != "" {
		tenantUUID, err := uuid.Parse(tenantInfo.TenantID)
		if err == nil {
			filter["tenant_id"] = tenantUUID
		}
	}
}

func (r *Repository[T]) applyQueryOptions(findOpts *options.FindOptions, opts ...*QueryOptions) {
	if len(opts) == 0 || opts[0] == nil {
		return
	}
	qo := opts[0]
	if qo.Sort != nil {
		findOpts.SetSort(qo.Sort)
	}
	if qo.Limit > 0 {
		findOpts.SetLimit(qo.Limit)
	}
	if qo.Skip > 0 {
		findOpts.SetSkip(qo.Skip)
	}
	if qo.Projection != nil {
		findOpts.SetProjection(qo.Projection)
	}
}

// --- Input sanitization (para uso em BindJSON/BindQuery, NÃO em filtros internos) ---

// InputSanitizer sanitiza input do usuário HTTP
type InputSanitizer struct {
	allowedFields map[string]bool
}

// NewInputSanitizer cria um sanitizador com campos permitidos customizáveis
func NewInputSanitizer(allowedFields ...string) *InputSanitizer {
	fields := map[string]bool{
		"_id":       true,
		"tenant_id": true,
		"name":      true,
		"email":     true,
		"status":    true,
		"active":    true,
	}
	for _, f := range allowedFields {
		fields[f] = true
	}
	return &InputSanitizer{allowedFields: fields}
}

// Sanitize sanitiza input do usuário
func (s *InputSanitizer) Sanitize(input map[string]interface{}) (map[string]interface{}, error) {
	if len(input) > 20 {
		return nil, fmt.Errorf("too many input fields")
	}

	sanitized := make(map[string]interface{})
	for key, value := range input {
		if !s.isValidField(key) {
			log.Printf("Invalid user input field rejected: %s", key)
			continue
		}
		sanitizedValue, err := sanitizeFilterValue(value)
		if err != nil {
			log.Printf("Invalid input value for field %s: %v", key, err)
			continue
		}
		sanitized[key] = sanitizedValue
	}

	return sanitized, nil
}

func (s *InputSanitizer) isValidField(fieldName string) bool {
	if s.allowedFields[fieldName] {
		return true
	}

	validPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]{0,50}$`)
	if !validPattern.MatchString(fieldName) {
		return false
	}

	dangerousPatterns := []string{"$", "javascript", "eval", "function", "where"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToLower(fieldName), pattern) {
			return false
		}
	}

	return true
}

func sanitizeFilterValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		if len(v) > 1000 {
			return nil, fmt.Errorf("string value too long")
		}
		dangerousPatterns := []string{"$", "javascript:", "eval(", "function("}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(v), pattern) {
				return nil, fmt.Errorf("dangerous pattern detected")
			}
		}
		return v, nil
	case int, int32, int64, float32, float64, bool:
		return v, nil
	case uuid.UUID:
		return v, nil
	case map[string]interface{}:
		return sanitizeNestedObject(v, 1)
	default:
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			if val.Len() > 100 {
				return nil, fmt.Errorf("array too large")
			}
		}
		return v, nil
	}
}

func sanitizeNestedObject(obj map[string]interface{}, depth int) (map[string]interface{}, error) {
	if depth > 3 {
		return nil, fmt.Errorf("object nesting too deep")
	}

	sanitized := make(map[string]interface{})
	for key, value := range obj {
		sanitizedValue, err := sanitizeFilterValue(value)
		if err != nil {
			continue
		}
		sanitized[key] = sanitizedValue
	}

	return sanitized, nil
}
