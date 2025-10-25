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

		// Extrai APENAS dados do Firebase (email/senha só tem UID e email)
		firebaseUID := token.UID
		email, _ := token.Claims["email"].(string)

		// Seta contexto do Gin - SEM tenant/user_id (dev deve setar no login)
		c.Set("auth_firebase_uid", firebaseUID)
		c.Set("auth_email", email)
		c.Set("auth_token", token)

		// Context básico
		ctx := context.WithValue(c.Request.Context(), "firebase_uid", firebaseUID)
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
		ID:          c.GetString("auth_user_id"),      // Setado pelo dev no login
		FirebaseUID: c.GetString("auth_firebase_uid"), // UID do Firebase
		Email:       c.GetString("auth_email"),        // Email do Firebase
		Name:        c.GetString("auth_name"),         // Setado pelo dev no login
		TenantID:    c.GetString("auth_tenant_id"),    // Setado pelo dev no login
	}
}

// HasRole verifica se usuário tem role específica
func (c *Context[T]) HasRole(role string) bool {
	userRole := c.GetString("auth_role")
	return userRole == role || userRole == "admin" // admin tem todas as roles
}

// RequireEmailVerified middleware para exigir email verificado (não aplicável para email/senha)
func RequireEmailVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Email/senha não tem verificação automática - sempre passa
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
	ID          string `json:"id"`           // Firebase UID ou ID customizado
	FirebaseUID string `json:"firebase_uid"` // UID do Firebase
	Email       string `json:"email"`        // Email do Firebase
	Name        string `json:"name"`         // Nome setado pelo dev
	TenantID    string `json:"tenant_id"`    // Tenant setado pelo dev
}
