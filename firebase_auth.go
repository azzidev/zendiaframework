package zendia

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type FirebaseAuthConfig struct {
	FirebaseClient *auth.Client
	PublicRoutes   []string
}

func (z *Zendia) SetupFirebaseAuth(config FirebaseAuthConfig) {
	z.firebaseAuthConfig = &config
	z.Use(z.firebaseAuthMiddleware())
}
func (z *Zendia) firebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if z.isFirebasePublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

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
		token, err := z.firebaseAuthConfig.FirebaseClient.VerifyIDToken(context.Background(), tokenString)
		if err != nil {
			c.JSON(401, gin.H{
				"success": false,
				"message": "Token inválido ou expirado",
			})
			c.Abort()
			return
		}

		firebaseUID := token.UID
		email, _ := token.Claims["email"].(string)

		c.Set("auth_firebase_uid", firebaseUID)
		c.Set("auth_email", email)
		c.Set("auth_token", token)

		if tenantID, ok := token.Claims["tenant_id"].(string); ok && tenantID != "" {
			c.Set("auth_tenant_id", tenantID)
			c.Header("X-Tenant-ID", tenantID)
		}
		if userID, ok := token.Claims["user_id"].(string); ok && userID != "" {
			c.Set("auth_user_id", userID)
			c.Set(UserIDKey, userID)
			c.Header("X-User-ID", userID)
		}
		if role, ok := token.Claims["role"].(string); ok && role != "" {
			c.Set("auth_role", role)
		}

		ctx := context.WithValue(c.Request.Context(), "firebase_uid", firebaseUID)
		ctx = context.WithValue(ctx, "email", email)
		if tenantID, exists := c.Get("auth_tenant_id"); exists {
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
		}
		if userID, exists := c.Get("auth_user_id"); exists {
			ctx = context.WithValue(ctx, UserIDKey, userID)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

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

func (c *Context[T]) GetAuthUser() *AuthUser {
	return &AuthUser{
		ID:          c.GetString("auth_user_id"),
		FirebaseUID: c.GetString("auth_firebase_uid"),
		Email:       c.GetString("auth_email"),
		Name:        c.GetString("auth_name"),
		TenantID:    c.GetString("auth_tenant_id"),
		Role:        c.GetString("auth_role"),
	}
}

func (c *Context[T]) HasRole(role string) bool {
	userRole := c.GetString("auth_role")
	return userRole == role || userRole == "admin"
}

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
			c.Next()
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

type AuthUser struct {
	ID          string `json:"id"`
	FirebaseUID string `json:"firebase_uid"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	TenantID    string `json:"tenant_id"`
	Role        string `json:"role"`
}
