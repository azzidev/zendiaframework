package zendia

// Firebase Custom Claims - Use essas constantes nos seus custom claims
const (
	// Custom Claims do Firebase (para SetCustomUserClaims)
	ClaimTenantID = "tenant_id" // ID do tenant no seu banco
	ClaimUserUUID = "user_uuid" // ID do usuário no seu banco (não usar "user_id" - é reservado)
	ClaimUserName = "user_name" // Nome do usuário
	ClaimRole     = "role"      // Role/perfil do usuário
)

// Context Keys - Internal context keys (do not modify)
const (
	// Gin Context Keys - used internally by framework
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

// Context Values - Values for context.Context (audit trail)
const (
	ContextFirebaseUID = "firebase_uid"
	ContextEmail       = "email"
	ContextTenantID    = "tenant_id"
	ContextUserID      = "user_id"
)

// Default Roles - Use these constants for common roles
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleViewer  = "viewer"
)

// Security Constants
const (
	MaxHeaderValueLength = 255
	MaxClaimValueLength  = 512
)



// Route Constants
const (
	RoutePublic    = "/public"
	RouteDocs      = "/docs"
	RouteAuth      = "/auth"
	RouteSwagger   = "/swagger"
	RouteHealth    = "/health"
	RouteAPIV1     = "/api/v1"
	RouteLogin     = "/auth/login"
	RouteMe        = "/me"
	RouteUsers     = "/users"
	RouteMetrics   = "/public/metrics"
	RouteTraces    = "/public/traces"
)

// Environment Variables
const (
	EnvGoogleCredentials = "GOOGLE_APPLICATION_CREDENTIALS"
	EnvFirebaseProjectID = "FIREBASE_PROJECT_ID"
)

// Default Values
const (
	DefaultCredentialsPath = "path/to/serviceAccountKey.json"
	DefaultMongoURI        = "mongodb://localhost:27017"
	DefaultPort            = ":8080"
	DefaultHost            = "localhost:8080"
	DefaultVersion         = "1.0"
	DefaultBasePath        = "/api/v1"
)

// Query Parameters
const (
	QuerySkip = "skip"
	QueryTake = "take"
	QueryName = "name"
)

// Field Names
const (
	FieldName     = "name"
	FieldTenantID = "tenant_id"
	FieldUserID   = "user_id"
)

// Messages
const (
	MsgLoginRequired           = "Faça login primeiro para setar o tenant"
	MsgInvalidPagination      = "Invalid pagination parameters"
	MsgInvalidUUID            = "Invalid UUID format"
	MsgRepositoryNotInit      = "Repository not properly initialized"
	MsgCreatedSuccess         = "Criado com sucesso."
	MsgUpdatedSuccess         = "Atualizado com sucesso."
	MsgRetrievedSuccess       = "Capturado com sucesso."
	MsgRetrievedByIDSuccess   = "Capturado com sucesso usando ID."
	MsgUserData               = "Dados do usuário"
	MsgMetricsFound           = "Metricas encontradas."
	MsgTracesFound            = "Traces encontradas."
	MsgLoginRealized          = "Login realizado"
	MsgCustomClaimsSet        = "Custom claims setados - token funciona para sempre"
	MsgTokenPlaceholder       = "firebase-token-with-custom-claims"
)
