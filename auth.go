package zendia

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// AuthConfig configuração de autenticação
type AuthConfig struct {
	FirebaseClient *auth.Client
	RequiredRoles  []string
	PublicRoutes   []string
}

// SetupAuth configura autenticação no framework
func (z *Zendia) SetupAuth(config AuthConfig) {
	z.authConfig = &config

	// Adiciona middleware de auth globalmente
	z.Use(z.authMiddleware())
}

// authMiddleware middleware interno de autenticação
func (z *Zendia) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verifica se é rota pública
		if z.isPublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Verifica token Firebase
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"success": false,
				"error":   "Authentication required",
				"code":    "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Valida token Firebase
		token, err := z.authConfig.FirebaseClient.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			c.JSON(401, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
				"code":    "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Extrai TODOS os dados do usuário
		firebaseUID := token.UID
		email, _ := token.Claims["email"].(string)
		name, _ := token.Claims["name"].(string)
		picture, _ := token.Claims["picture"].(string)
		emailVerified, _ := token.Claims["email_verified"].(bool)
		role, _ := token.Claims["role"].(string)
		tenantID, _ := token.Claims["tenant_id"].(string)

		// Adiciona TUDO ao contexto do Gin
		c.Set("auth_firebase_uid", firebaseUID)
		c.Set("auth_user_id", firebaseUID) // Por enquanto usa Firebase UID
		c.Set("auth_email", email)
		c.Set("auth_name", name)
		c.Set("auth_picture", picture)
		c.Set("auth_email_verified", emailVerified)
		c.Set("auth_role", role)
		c.Set("auth_tenant_id", tenantID)
		c.Set("auth_token", token)
		c.Set("firebase_claims", token.Claims)

		// Para o TenantMiddleware do framework usar
		if tenantID != "" {
			c.Header("X-Tenant-ID", tenantID)
		}
		c.Header("X-User-ID", firebaseUID)

		// TODO: Buscar usuário no banco pelo email para pegar UUID correto
		// Por enquanto usa Firebase UID
		ctx := context.WithValue(c.Request.Context(), "user_id", firebaseUID)
		ctx = context.WithValue(ctx, "email", email)
		ctx = context.WithValue(ctx, "tenant_id", tenantID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// isPublicRoute verifica se a rota é pública
func (z *Zendia) isPublicRoute(path string) bool {
	if z.authConfig == nil {
		return true
	}

	publicRoutes := []string{"/health"}
	publicRoutes = append(publicRoutes, z.authConfig.PublicRoutes...)

	for _, route := range publicRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}
	return false
}
