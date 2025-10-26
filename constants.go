package zendia

// Firebase Custom Claims - Use essas constantes nos seus custom claims
const (
	// Custom Claims do Firebase (para SetCustomUserClaims)
	ClaimTenantID = "tenant_id"   // ID do tenant no seu banco
	ClaimUserUUID = "user_uuid"   // ID do usuário no seu banco (não usar "user_id" - é reservado)
	ClaimUserName = "user_name"   // Nome do usuário
	ClaimRole     = "role"        // Role/perfil do usuário
)

// Context Keys - Chaves internas do contexto (não altere)
const (
	// Gin Context Keys - usadas internamente pelo framework
	AuthFirebaseUIDKey = "auth_firebase_uid"
	AuthEmailKey       = "auth_email"
	AuthTokenKey       = "auth_token"
	AuthTenantIDKey    = "auth_tenant_id"
	AuthUserIDKey      = "auth_user_id"
	AuthRoleKey        = "auth_role"
	AuthNameKey        = "auth_name"
)

// HTTP Headers - Headers automáticos do framework
const (
	HeaderTenantID = "X-Tenant-ID"
	HeaderUserID   = "X-User-ID"
	HeaderUserName = "X-User-Name"
)

// Default Public Routes - Rotas públicas padrão do Firebase Auth
var DefaultPublicRoutes = []string{
	"/health",
	"/docs", 
	"/swagger",
}

// Response Fields - Campos padrão das respostas JSON
const (
	ResponseSuccess = "success"
	ResponseMessage = "message"
	ResponseData    = "data"
	ResponseError   = "error"
)

// Context Values - Valores do context.Context (auditoria)
const (
	ContextFirebaseUID = "firebase_uid"
	ContextEmail       = "email"
	ContextTenantID    = "tenant_id"
	ContextUserID      = "user_id"
)

// Roles padrão - Use essas constantes para roles comuns
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleViewer  = "viewer"
)