package zendia

import (
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// GetAuthUser retorna dados do usuário autenticado
func GetAuthUser(c *gin.Context) *AuthUser {
	return &AuthUser{
		ID:            c.GetString("auth_user_id"),
		Email:         c.GetString("auth_email"),
		Name:          c.GetString("auth_name"),
		Picture:       c.GetString("auth_picture"),
		EmailVerified: c.GetBool("auth_email_verified"),
		Role:          c.GetString("auth_role"),
		TenantID:      c.GetString("auth_tenant_id"),
	}
}

// GetFirebaseToken retorna o token Firebase completo
func GetFirebaseToken(c *gin.Context) *auth.Token {
	if token, exists := c.Get("auth_token"); exists {
		return token.(*auth.Token)
	}
	return nil
}

// GetFirebaseClaims retorna todos os claims do token
func GetFirebaseClaims(c *gin.Context) map[string]interface{} {
	if claims, exists := c.Get("firebase_claims"); exists {
		return claims.(map[string]interface{})
	}
	return nil
}

// IsAuthenticated verifica se o usuário está autenticado
func IsAuthenticated(c *gin.Context) bool {
	return c.GetString("auth_user_id") != ""
}

// HasRole verifica se o usuário tem uma role específica
func HasRole(c *gin.Context, role string) bool {
	userRole := c.GetString("auth_role")
	return userRole == role || userRole == "admin"
}

// HasAnyRole verifica se o usuário tem qualquer uma das roles
func HasAnyRole(c *gin.Context, roles ...string) bool {
	userRole := c.GetString("auth_role")
	if userRole == "admin" {
		return true
	}
	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// AuthUser estrutura com dados do usuário autenticado
type AuthUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role"`
	TenantID      string `json:"tenant_id"`
}

// Context helpers para usar nos handlers
func (c *Context[T]) GetAuthUser() *AuthUser {
	return GetAuthUser(c.Context)
}

func (c *Context[T]) IsAuthenticated() bool {
	return IsAuthenticated(c.Context)
}

func (c *Context[T]) HasRole(role string) bool {
	return HasRole(c.Context, role)
}

func (c *Context[T]) GetFirebaseToken() *auth.Token {
	return GetFirebaseToken(c.Context)
}