package zendia

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoMetricsPersister implementação MongoDB para persistência de métricas
type MongoMetricsPersister struct {
	collection *mongo.Collection
}

// NewMongoMetricsPersister cria um novo persistidor MongoDB
func NewMongoMetricsPersister(collection *mongo.Collection) *MongoMetricsPersister {
	return &MongoMetricsPersister{
		collection: collection,
	}
}

// Save salva snapshot de métricas no MongoDB
func (mp *MongoMetricsPersister) Save(snapshot MetricsSnapshot) error {
	if mp.collection == nil {
		return fmt.Errorf("collection is nil")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Validação básica do snapshot
	if snapshot.ID == "" {
		snapshot.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now()
	}
	if snapshot.Endpoints == nil {
		snapshot.Endpoints = make(map[string]interface{})
	}
	if snapshot.MemoryUsage == nil {
		snapshot.MemoryUsage = make(map[string]interface{})
	}

	_, err := mp.collection.InsertOne(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert metrics snapshot: %w", err)
	}
	return nil
}

// GetHistory busca histórico de métricas por período
func (mp *MongoMetricsPersister) GetHistory(tenantID string, from, to time.Time) ([]MetricsSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"timestamp": bson.M{
			"$gte": from,
			"$lte": to,
		},
	}

	// Adiciona filtro por tenant se fornecido
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}

	// Ordena por timestamp (mais recente primeiro)
	opts := options.Find().SetSort(bson.D{{"timestamp", -1}})

	cursor, err := mp.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var snapshots []MetricsSnapshot
	if err = cursor.All(ctx, &snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// GetLatest busca as métricas mais recentes
func (mp *MongoMetricsPersister) GetLatest(tenantID string, limit int) ([]MetricsSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}

	opts := options.Find().
		SetSort(bson.D{{"timestamp", -1}}).
		SetLimit(int64(limit))

	cursor, err := mp.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var snapshots []MetricsSnapshot
	if err = cursor.All(ctx, &snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Cleanup remove métricas antigas (mais de X dias)
func (mp *MongoMetricsPersister) Cleanup(olderThanDays int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	filter := bson.M{
		"timestamp": bson.M{"$lt": cutoff},
	}

	_, err := mp.collection.DeleteMany(ctx, filter)
	return err
}

// CreateIndexes cria índices otimizados para consultas
func (mp *MongoMetricsPersister) CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"timestamp", -1}},
			Options: options.Index().SetName("timestamp_desc"),
		},
		{
			Keys: bson.D{{"tenant_id", 1}, {"timestamp", -1}},
			Options: options.Index().SetName("tenant_timestamp"),
		},
		{
			Keys: bson.D{{"timestamp", 1}},
			Options: options.Index().
				SetName("timestamp_ttl").
				SetExpireAfterSeconds(30 * 24 * 60 * 60), // 30 dias TTL
		},
	}

	_, err := mp.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// GetAggregatedStats retorna estatísticas agregadas por período
func (mp *MongoMetricsPersister) GetAggregatedStats(tenantID string, from, to time.Time, interval string) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define o formato de agrupamento baseado no intervalo
	var dateFormat string
	switch interval {
	case "hour":
		dateFormat = "%Y-%m-%d-%H"
	case "day":
		dateFormat = "%Y-%m-%d"
	case "month":
		dateFormat = "%Y-%m"
	default:
		dateFormat = "%Y-%m-%d-%H" // padrão: por hora
	}

	matchStage := bson.M{
		"$match": bson.M{
			"timestamp": bson.M{
				"$gte": from,
				"$lte": to,
			},
		},
	}

	if tenantID != "" {
		matchStage["$match"].(bson.M)["tenant_id"] = tenantID
	}

	pipeline := []bson.M{
		matchStage,
		{
			"$group": bson.M{
				"_id": bson.M{
					"$dateToString": bson.M{
						"format": dateFormat,
						"date":   "$timestamp",
					},
				},
				"avg_requests":       bson.M{"$avg": "$total_requests"},
				"avg_errors":         bson.M{"$avg": "$total_errors"},
				"avg_error_rate":     bson.M{"$avg": "$error_rate"},
				"avg_active_requests": bson.M{"$avg": "$active_requests"},
				"count":              bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"_id": 1},
		},
	}

	cursor, err := mp.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}