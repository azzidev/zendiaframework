package zendia

import (
	"context"
	"html"
	"log"
	"regexp"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthConfig configuração para autenticação Firebase
type FirebaseAuthConfig struct {
	FirebaseClient *auth.Client
	PublicRoutes   []string
}

// SetupFirebaseAuth configura autenticação Firebase no framework
func (z *Zendia) SetupFirebaseAuth(config FirebaseAuthConfig) {
	z.firebaseAuthConfig = &config
	z.Use(z.firebaseAuthMiddleware())
}

// firebaseAuthMiddleware middleware para validação de tokens Firebase
func (z *Zendia) firebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if z.isFirebasePublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(NewUnauthorizedError("Token de autenticação obrigatório"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := z.firebaseAuthConfig.FirebaseClient.VerifyIDToken(c.Request.Context(), tokenString)
		if err != nil {
			log.Printf("Firebase token verification failed: %v", err)
			c.Error(NewUnauthorizedError("Token inválido ou expirado"))
			c.Abort()
			return
		}

		firebaseUID := token.UID
		email, _ := token.Claims["email"].(string)

		c.Set(AuthFirebaseUIDKey, firebaseUID)
		c.Set(AuthEmailKey, email)
		c.Set(AuthTokenKey, token)

		if tenantID, ok := token.Claims[ClaimTenantID].(string); ok && tenantID != "" {
			if sanitizedTenantID := sanitizeHeaderValue(tenantID); sanitizedTenantID != "" {
				c.Set(AuthTenantIDKey, sanitizedTenantID)
				c.Header(HeaderTenantID, sanitizedTenantID)
			}
		}
		if userID, ok := token.Claims[ClaimUserUUID].(string); ok && userID != "" {
			if sanitizedUserID := sanitizeHeaderValue(userID); sanitizedUserID != "" {
				c.Set(AuthUserIDKey, sanitizedUserID)
				c.Set(UserIDKey, sanitizedUserID)
				c.Header(HeaderUserID, sanitizedUserID)
			}
		}

		if name, ok := token.Claims[ClaimUserName].(string); ok && name != "" {
			if sanitizedName := sanitizeHeaderValue(name); sanitizedName != "" {
				c.Set(AuthNameKey, sanitizedName)
				c.Set(UserNameKey, sanitizedName)
				c.Header(HeaderUserName, sanitizedName)
			}
		}

		ctx := context.WithValue(c.Request.Context(), ContextFirebaseUID, firebaseUID)
		ctx = context.WithValue(ctx, ContextEmail, email)
		if tenantID, exists := c.Get(AuthTenantIDKey); exists {
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
		}
		if userID, exists := c.Get(AuthUserIDKey); exists {
			ctx = context.WithValue(ctx, UserIDKey, userID)
		}
		if userName, exists := c.Get(AuthNameKey); exists {
			ctx = context.WithValue(ctx, UserNameKey, userName)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// isFirebasePublicRoute verifica se a rota é pública (não precisa de auth)
func (z *Zendia) isFirebasePublicRoute(path string) bool {
	if z.firebaseAuthConfig == nil {
		return true
	}

	publicRoutes := []string{"/health", "/docs", "/swagger"}
	publicRoutes = append(publicRoutes, z.firebaseAuthConfig.PublicRoutes...)

	for _, route := range publicRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}
	return false
}

// GetAuthUser retorna informações do usuário autenticado
func (c *Context[T]) GetAuthUser() *AuthUser {
	return &AuthUser{
		ID:          c.GetString(AuthUserIDKey),
		FirebaseUID: c.GetString(AuthFirebaseUIDKey),
		Email:       c.GetString(AuthEmailKey),
		Name:        c.GetString(AuthNameKey),
		TenantID:    c.GetString(AuthTenantIDKey),
	}
}

// sanitizeHeaderValue sanitiza valores de header para prevenir XSS
func sanitizeHeaderValue(value string) string {
	value = html.EscapeString(value)
	re := regexp.MustCompile(`[\r\n]`)
	value = re.ReplaceAllString(value, "")

	if len(value) > 255 {
		value = value[:255]
	}
	return strings.TrimSpace(value)
}

// AuthUser representa um usuário autenticado
type AuthUser struct {
	ID          string `json:"id"`
	FirebaseUID string `json:"firebase_uid"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	TenantID    string `json:"tenant_id"`
}
