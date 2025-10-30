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
		// Use request context instead of Background
		token, err := z.firebaseAuthConfig.FirebaseClient.VerifyIDToken(c.Request.Context(), tokenString)
		if err != nil {
			log.Printf("Firebase token verification failed: %v", err)
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

		// Sanitize and validate claims before setting headers
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
		if role, ok := token.Claims[ClaimRole].(string); ok && role != "" {
			if sanitizedRole := sanitizeHeaderValue(role); sanitizedRole != "" {
				c.Set(AuthRoleKey, sanitizedRole)
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

// sanitizeHeaderValue prevents XSS by sanitizing header values
func sanitizeHeaderValue(value string) string {
	// Remove any control characters and HTML entities
	value = html.EscapeString(value)
	// Remove newlines and carriage returns to prevent header injection
	re := regexp.MustCompile(`[\r\n]`)
	value = re.ReplaceAllString(value, "")
	// Limit length to prevent DoS
	if len(value) > 255 {
		value = value[:255]
	}
	return strings.TrimSpace(value)
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(AuthRoleKey)
		if !exists {
			log.Printf("Role not found for request: %s", c.Request.URL.Path)
			c.JSON(403, gin.H{
				"success": false,
				"message": "Role não encontrada",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			log.Printf("Invalid role type for request: %s", c.Request.URL.Path)
			c.JSON(403, gin.H{
				"success": false,
				"message": "Role inválida",
			})
			c.Abort()
			return
		}

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

		log.Printf("Insufficient permissions for user role '%s' on path: %s", role, c.Request.URL.Path)
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
