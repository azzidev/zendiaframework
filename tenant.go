package zendia

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// TenantContext chaves para contexto de tenant
const (
	TenantIDKey = "tenant_id"
	UserIDKey   = "user_id"
	ActionAtKey = "action_at"
)

// TenantInfo informações do tenant no contexto
type TenantInfo struct {
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	ActionAt time.Time `json:"actionAt"`
}

// TenantExtractor função para extrair informações do tenant
type TenantExtractor func(*gin.Context) TenantInfo

// DefaultTenantExtractor extrator padrão que busca nos headers
func DefaultTenantExtractor(c *gin.Context) TenantInfo {
	return TenantInfo{
		TenantID: c.GetHeader("X-Tenant-ID"),
		UserID:   c.GetHeader("X-User-ID"),
		ActionAt: time.Now(),
	}
}

// TenantMiddleware middleware para carregar contexto do tenant
func TenantMiddleware(extractor TenantExtractor) gin.HandlerFunc {
	if extractor == nil {
		extractor = DefaultTenantExtractor
	}
	
	return func(c *gin.Context) {
		tenantInfo := extractor(c)
		
		// Adiciona ao contexto do Gin
		c.Set(TenantIDKey, tenantInfo.TenantID)
		c.Set(UserIDKey, tenantInfo.UserID)
		c.Set(ActionAtKey, tenantInfo.ActionAt)
		
		// Cria contexto com informações do tenant
		ctx := context.WithValue(c.Request.Context(), TenantIDKey, tenantInfo.TenantID)
		ctx = context.WithValue(ctx, UserIDKey, tenantInfo.UserID)
		ctx = context.WithValue(ctx, ActionAtKey, tenantInfo.ActionAt)
		
		// Atualiza o request com o novo contexto
		c.Request = c.Request.WithContext(ctx)
		
		c.Next()
	}
}

// GetTenantID obtém o tenant ID do contexto
func GetTenantID(ctx context.Context) string {
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok {
		return tenantID
	}
	return ""
}

// GetUserID obtém o user ID do contexto
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetActionAt obtém o timestamp da ação do contexto
func GetActionAt(ctx context.Context) time.Time {
	if actionAt, ok := ctx.Value(ActionAtKey).(time.Time); ok {
		return actionAt
	}
	return time.Now()
}

// GetTenantInfo obtém todas as informações do tenant do contexto
func GetTenantInfo(ctx context.Context) TenantInfo {
	return TenantInfo{
		TenantID: GetTenantID(ctx),
		UserID:   GetUserID(ctx),
		ActionAt: GetActionAt(ctx),
	}
}

// GetTenantIDFromGin obtém tenant ID do gin.Context
func GetTenantIDFromGin(c *gin.Context) string {
	if tenantID, exists := c.Get(TenantIDKey); exists {
		return tenantID.(string)
	}
	return ""
}

// GetUserIDFromGin obtém user ID do gin.Context
func GetUserIDFromGin(c *gin.Context) string {
	if userID, exists := c.Get(UserIDKey); exists {
		return userID.(string)
	}
	return ""
}

// GetActionAtFromGin obtém action timestamp do gin.Context
func GetActionAtFromGin(c *gin.Context) time.Time {
	if actionAt, exists := c.Get(ActionAtKey); exists {
		return actionAt.(time.Time)
	}
	return time.Now()
}

// GetTenantInfoFromGin obtém informações do tenant do gin.Context
func GetTenantInfoFromGin(c *gin.Context) TenantInfo {
	return TenantInfo{
		TenantID: GetTenantIDFromGin(c),
		UserID:   GetUserIDFromGin(c),
		ActionAt: GetActionAtFromGin(c),
	}
}