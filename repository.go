package zendia

import (
	"time"

	"github.com/google/uuid"
)

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

// MongoAuditableEntity interface para entidades MongoDB
type MongoAuditableEntity interface {
	GetID() uuid.UUID
	SetID(uuid.UUID)
	SetTenantID(string)
}

// QueryOptions opções para queries
type QueryOptions struct {
	Sort       map[string]interface{}
	Limit      int64
	Skip       int64
	Projection map[string]interface{}
}

type Order struct {
	By string
	At int64
}

type Pagination struct {
	Skip int
	Take int
}

type Between struct {
	Type  string
	Start time.Time
	End   time.Time
}
