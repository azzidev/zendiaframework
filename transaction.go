package zendia

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// TxFunc é a função executada dentro de uma transação
type TxFunc func(ctx context.Context) error

// RunTransaction executa uma função dentro de uma transação MongoDB.
func RunTransaction(ctx context.Context, client *mongo.Client, fn TxFunc) error {
	session, err := client.StartSession()
	if err != nil {
		return NewInternalError("Failed to start transaction session: " + err.Error())
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Propaga os valores de tenant/user do ctx original para o sessCtx
		// Necessário porque o mongo.SessionContext não herda os valores do context pai
		merged := propagateTenantContext(ctx, sessCtx)
		return nil, fn(merged)
	})
	return err
}

// propagateTenantContext copia os valores de tenant/user do src para o dst.
func propagateTenantContext(src context.Context, dst context.Context) context.Context {
	for _, key := range []string{TenantIDKey, UserIDKey, UserNameKey, ActionAtKey, ContextEmail} {
		if val := src.Value(key); val != nil {
			dst = context.WithValue(dst, key, val)
		}
	}
	return dst
}
