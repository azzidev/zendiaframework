package zendia

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// StreamMessage representa uma mensagem do stream com metadata
type StreamMessage struct {
	TenantID string
	UserID   string
	UserName string
	Payload  []byte
}

// StreamHandler função que processa uma mensagem do stream
type StreamHandler func(ctx context.Context, payload []byte) error

// StreamConsumerConfig configuração do consumer
type StreamConsumerConfig struct {
	Stream   string
	Group    string
	Consumer string
	Handler  StreamHandler
}

// StreamClient client de Redis Stream com gerenciamento automático de tenant
type StreamClient struct {
	redis *redis.Client
}

// NewStreamClient cria um novo client de Redis Stream
func NewStreamClient(redisClient *redis.Client) *StreamClient {
	return &StreamClient{redis: redisClient}
}

// Publish publica uma mensagem no stream com tenant_id automático do context
func (sc *StreamClient) Publish(ctx context.Context, stream string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	values := map[string]interface{}{
		"payload": string(data),
	}

	// Injeta metadata do context automaticamente
	if tenantID := GetTenantID(ctx); tenantID != "" {
		values["tenant_id"] = tenantID
	}
	if userID := GetUserID(ctx); userID != "" {
		values["user_id"] = userID
	}
	if userName := GetUserName(ctx); userName != "" {
		values["user_name"] = userName
	}

	return sc.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}

// Subscribe registra um consumer para um stream
// O handler recebe um context já com tenant_id, user_id e user_name injetados
func (sc *StreamClient) Subscribe(ctx context.Context, config StreamConsumerConfig) {
	sc.ensureGroup(ctx, config.Stream, config.Group)

	log.Printf("🎧 Listening to stream: %s (group: %s, consumer: %s)", config.Stream, config.Group, config.Consumer)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streams, err := sc.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    config.Group,
					Consumer: config.Consumer,
					Streams:  []string{config.Stream, ">"},
					Count:    10,
					Block:    0,
				}).Result()
				if err != nil {
					log.Printf("❌ Stream error [%s]: %v", config.Stream, err)
					time.Sleep(1 * time.Second)
					continue
				}

				for _, stream := range streams {
					for _, msg := range stream.Messages {
						sc.processMessage(ctx, config, msg)
					}
				}
			}
		}
	}()
}

func (sc *StreamClient) processMessage(ctx context.Context, config StreamConsumerConfig, msg redis.XMessage) {
	// Extrai metadata e injeta no context da mesma forma que o firebase_auth middleware
	msgCtx := ctx
	if tenantID, ok := msg.Values["tenant_id"].(string); ok && tenantID != "" {
		msgCtx = context.WithValue(msgCtx, TenantIDKey, tenantID)
	}
	if userID, ok := msg.Values["user_id"].(string); ok && userID != "" {
		msgCtx = context.WithValue(msgCtx, UserIDKey, userID)
	}
	if userName, ok := msg.Values["user_name"].(string); ok && userName != "" {
		msgCtx = context.WithValue(msgCtx, UserNameKey, userName)
	}
	msgCtx = context.WithValue(msgCtx, ActionAtKey, time.Now())

	// Extrai payload
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		log.Printf("❌ Invalid payload in message %s", msg.ID)
		return
	}

	// Chama o handler
	if err := config.Handler(msgCtx, []byte(payload)); err != nil {
		log.Printf("❌ Handler error [%s]: %v", config.Stream, err)
		return
	}

	// ACK automático
	sc.redis.XAck(ctx, config.Stream, config.Group, msg.ID)
}

func (sc *StreamClient) ensureGroup(ctx context.Context, stream, group string) {
	err := sc.redis.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("⚠️  Could not create consumer group [%s/%s]: %v", stream, group, err)
	}
}
