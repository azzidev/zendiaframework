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

		c.Set(AuthFirebaseUIDKey, firebaseUID)
		c.Set(AuthEmailKey, email)
		c.Set(AuthTokenKey, token)

		if tenantID, ok := token.Claims[ClaimTenantID].(string); ok && tenantID != "" {
			c.Set(AuthTenantIDKey, tenantID)
			c.Header(HeaderTenantID, tenantID)
		}
		if userID, ok := token.Claims[ClaimUserUUID].(string); ok && userID != "" {
			c.Set(AuthUserIDKey, userID)
			c.Set(UserIDKey, userID)
			c.Header(HeaderUserID, userID)
		}
		if role, ok := token.Claims[ClaimRole].(string); ok && role != "" {
			c.Set(AuthRoleKey, role)
		}
		if name, ok := token.Claims[ClaimUserName].(string); ok && name != "" {
			c.Set(AuthNameKey, name)
		}

		ctx := context.WithValue(c.Request.Context(), ContextFirebaseUID, firebaseUID)
		ctx = context.WithValue(ctx, ContextEmail, email)
		if tenantID, exists := c.Get(AuthTenantIDKey); exists {
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
		}
		if userID, exists := c.Get(AuthUserIDKey); exists {
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

	publicRoutes := DefaultPublicRoutes
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
		ID:          c.GetString(AuthUserIDKey),
		FirebaseUID: c.GetString(AuthFirebaseUIDKey),
		Email:       c.GetString(AuthEmailKey),
		Name:        c.GetString(AuthNameKey),
		TenantID:    c.GetString(AuthTenantIDKey),
		Role:        c.GetString(AuthRoleKey),
	}
}

func (c *Context[T]) HasRole(role string) bool {
	userRole := c.GetString(AuthRoleKey)
	return userRole == role || userRole == RoleAdmin
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(AuthRoleKey)
		if !exists {
			c.JSON(403, gin.H{
				"success": false,
				"message": "Role não encontrada",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		if role == RoleAdmin {
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
