package zendia

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthConfig configuração específica do Firebase
type FirebaseAuthConfig struct {
	FirebaseClient *auth.Client
	PublicRoutes   []string
}

// SetupFirebaseAuth configura autenticação Firebase
func (z *Zendia) SetupFirebaseAuth(config FirebaseAuthConfig) {
	z.firebaseAuthConfig = &config
	z.Use(z.firebaseAuthMiddleware())
}

// firebaseAuthMiddleware middleware Firebase - só autentica, não resolve dados
func (z *Zendia) firebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verifica se é rota pública
		if z.isFirebasePublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Verifica token Firebase
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"success": false,
				"message": "Token de autenticação obrigatório",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Valida token Firebase
		token, err := z.firebaseAuthConfig.FirebaseClient.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			c.JSON(401, gin.H{
				"success": false,
				"message": "Token inválido ou expirado",
			})
			c.Abort()
			return
		}

		// Extrai APENAS dados do Firebase (não assume banco)
		firebaseUID := token.UID
		email, _ := token.Claims["email"].(string)
		name, _ := token.Claims["name"].(string)
		picture, _ := token.Claims["picture"].(string)
		emailVerified, _ := token.Claims["email_verified"].(bool)

		// Seta contexto do Gin - SEM tenant (dev deve setar)
		c.Set("auth_firebase_uid", firebaseUID)
		c.Set("auth_user_id", firebaseUID) // Default: Firebase UID
		c.Set("auth_email", email)
		c.Set("auth_name", name)
		c.Set("auth_picture", picture)
		c.Set("auth_email_verified", emailVerified)
		c.Set("auth_token", token)

		// Headers básicos
		c.Header("X-User-ID", firebaseUID)

		// Context básico
		ctx := context.WithValue(c.Request.Context(), "user_id", firebaseUID)
		ctx = context.WithValue(ctx, "email", email)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// isFirebasePublicRoute verifica se a rota é pública
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

// GetAuthUser retorna dados do usuário autenticado
func (c *Context[T]) GetAuthUser() *AuthUser {
	return &AuthUser{
		ID:            c.GetString("auth_user_id"),
		FirebaseUID:   c.GetString("auth_firebase_uid"),
		Email:         c.GetString("auth_email"),
		Name:          c.GetString("auth_name"),
		Picture:       c.GetString("auth_picture"),
		EmailVerified: c.GetBool("auth_email_verified"),
		TenantID:      c.GetString("auth_tenant_id"),
		Role:          c.GetString("auth_role"),
	}
}

// HasRole verifica se usuário tem role específica
func (c *Context[T]) HasRole(role string) bool {
	userRole := c.GetString("auth_role")
	return userRole == role || userRole == "admin" // admin tem todas as roles
}

// RequireEmailVerified middleware para exigir email verificado
func RequireEmailVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		emailVerified, exists := c.Get("auth_email_verified")
		if !exists || !emailVerified.(bool) {
			c.JSON(403, gin.H{
				"success": false,
				"message": "Email não verificado",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole middleware para exigir role específica
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("auth_role")
		if !exists {
			c.JSON(403, gin.H{
				"success": false,
				"message": "Role não encontrada",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		if role == "admin" {
			c.Next() // Admin tem acesso a tudo
			return
		}

		for _, requiredRole := range roles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(403, gin.H{
			"success": false,
			"message": "Permissão insuficiente",
		})
		c.Abort()
	}
}

// AuthUser representa dados do usuário autenticado
type AuthUser struct {
	ID            string `json:"id"`
	FirebaseUID   string `json:"firebase_uid"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	EmailVerified bool   `json:"email_verified"`
	TenantID      string `json:"tenant_id"`
	Role          string `json:"role"`
}
