package zendia

import "time"

// Firebase Custom Claims - Use essas constantes nos seus custom claims
const (
	// Custom Claims do Firebase (para SetCustomUserClaims)
	ClaimTenantID string = "tenant_id" // ID do tenant no seu banco
	ClaimUserUUID string = "user_uuid" // ID do usuário no seu banco (não usar "user_id" - é reservado)
	ClaimUserName string = "user_name" // Nome do usuário

)

// Context Keys - Internal context keys (do not modify)
const (
	// Gin Context Keys - used internally by framework
	AuthFirebaseUIDKey string = "auth_firebase_uid"
	AuthEmailKey       string = "auth_email"
	AuthTokenKey       string = "auth_token"
	AuthTenantIDKey    string = "auth_tenant_id"
	AuthUserIDKey      string = "auth_user_id"
	AuthNameKey        string = "auth_name"
)

// HTTP Headers - Headers automáticos do framework
const (
	HeaderTenantID string = "X-Tenant-ID"
	HeaderUserID   string = "X-User-ID"
	HeaderUserName string = "X-User-Name"
)

// Default Public Routes - Rotas públicas padrão do Firebase Auth
var DefaultPublicRoutes = []string{
	"/health",
	"/docs",
	"/swagger",
}

// Response Fields - Campos padrão das respostas JSON
const (
	ResponseSuccess string = "success"
	ResponseMessage string = "message"
	ResponseData    string = "data"
	ResponseError   string = "error"
)

// Context Values - Values for context.Context (audit trail)
const (
	ContextFirebaseUID string = "firebase_uid"
	ContextEmail       string = "email"
	ContextTenantID    string = "tenant_id"
	ContextUserID      string = "user_id"
)

// Security Constants
const (
	MaxHeaderValueLength = 255
	MaxClaimValueLength  = 512
)

// Route Constants
const (
	RoutePublic  = "/public"
	RouteDocs    = "/docs"
	RouteAuth    = "/auth"
	RouteSwagger = "/swagger"
	RouteHealth  = "/health"
	RouteAPIV1   = "/api/v1"
	RouteLogin   = "/auth/login"
	RouteMe      = "/me"
	RouteUsers   = "/users"
	RouteMetrics = "/public/metrics"
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

// Cache Constants
const (
	DefaultCacheTTL          = 10 * time.Minute
	DefaultCacheMaxSize      = 10000
	DefaultMemoryCacheMaxMem = 5 * 1024 * 1024   // 5MB (in-memory)
	DefaultRedisCacheMaxMem  = 100 * 1024 * 1024 // 100MB (Redis)
	DefaultCacheKeyPrefix    = "zendia:"
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
	MsgLoginRequired        = "Faça login primeiro para setar o tenant"
	MsgInvalidPagination    = "Invalid pagination parameters"
	MsgInvalidUUID          = "Invalid UUID format"
	MsgRepositoryNotInit    = "Repository not properly initialized"
	MsgCreatedSuccess       = "Criado com sucesso."
	MsgUpdatedSuccess       = "Atualizado com sucesso."
	MsgRetrievedSuccess     = "Capturado com sucesso."
	MsgRetrievedByIDSuccess = "Capturado com sucesso usando ID."
	MsgUserData             = "Dados do usuário"
	MsgMetricsFound         = "Metricas encontradas."

	MsgLoginRealized    = "Login realizado"
	MsgCustomClaimsSet  = "Custom claims setados - token funciona para sempre"
	MsgTokenPlaceholder = "firebase-token-with-custom-claims"
)
