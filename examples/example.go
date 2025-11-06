package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	zendia "github.com/azzidev/zendiaframework"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
)

// User entidade completa com auditoria e tenant usando UUID nativo
type User struct {
	ID       uuid.UUID        `bson:"_id" json:"id"`
	Name     string           `bson:"name" json:"name" validate:"required,min=2,max=50"`
	Email    string           `bson:"email" json:"email" validate:"required,email"`
	Age      int              `bson:"age" json:"age" validate:"gte=0,lte=120"`
	TenantID uuid.UUID        `bson:"tenant_id" json:"tenant_id"`
	Created  zendia.AuditInfo `bson:"created" json:"created"`
	Updated  zendia.AuditInfo `bson:"updated" json:"updated"`
	Deleted  zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"`
	Active   bool             `bson:"active" json:"active"`
}

// Implementa MongoAuditableEntity e AuditableEntity
func (u *User) GetID() uuid.UUID                        { return u.ID }
func (u *User) SetID(id uuid.UUID)                      { u.ID = id }
func (u *User) SetCreated(info zendia.AuditInfo)        { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo)        { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo)        { u.Deleted = info }
func (u *User) SetActive(active bool)                   { u.Active = active }
func (u *User) SetTenantID(s string) {
	if s != "" {
		u.TenantID = uuid.MustParse(s)
	}
}

func main() {
	// Initialize Firebase with proper error handling
	ctx := context.Background()
	// Use environment variable or fallback
	credentialsPath := zendia.DefaultCredentialsPath
	if envPath := os.Getenv(zendia.EnvGoogleCredentials); envPath != "" {
		credentialsPath = envPath
	}

	opt := option.WithCredentialsFile(credentialsPath)
	firebaseApp, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("Firebase init failed:", err)
	}
	firebaseAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatal("Firebase Auth init failed:", err)
	}

	app := zendia.New()

	// Middlewares
	app.Use(zendia.Logger())
	app.Use(zendia.CORS("*"))

	// Setup Firebase Auth - extrai custom claims automaticamente
	app.SetupFirebaseAuth(zendia.FirebaseAuthConfig{
		FirebaseClient: firebaseAuth,
		PublicRoutes:   []string{zendia.RoutePublic, zendia.RouteDocs, zendia.RouteAuth},
	})

	// Rota de login PÚBLICA: email/senha → Firebase token + custom claims
	app.POST(zendia.RouteLogin, zendia.Handle(func(c *zendia.Context[any]) error {
		var req struct {
			Email    string `json:"email" validate:"required,email,max=255"`
			Password string `json:"password" validate:"required,min=8,max=128"`
		}
		if err := c.Context.ShouldBindJSON(&req); err != nil {
			return err
		}

		// 1. Autentica no Firebase (REST API)
		// token, err := authenticateFirebase(req.Email, req.Password)
		// if err != nil { return zendia.NewUnauthorizedError("Credenciais inválidas") }

		// 2. Decodifica token para pegar Firebase UID
		// decodedToken, err := firebaseAuth.VerifyIDToken(ctx, token)
		// if err != nil { return zendia.NewUnauthorizedError("Token inválido") }

		// 3. Busca dados do SEU banco
		// userFromDB := myUserRepo.FindByEmail(req.Email)

		// 4. Seta custom claims (PARA SEMPRE) - USE AS CONSTANTES!
		// claims := map[string]interface{}{
		//     zendia.ClaimTenantID: userFromDB.TenantID,
		//     zendia.ClaimUserUUID: userFromDB.ID,
		//     zendia.ClaimUserName: userFromDB.Name,

		// }
		// err = firebaseAuth.SetCustomUserClaims(ctx, decodedToken.UID, claims)

		c.Success(zendia.MsgLoginRealized, map[string]interface{}{
			zendia.ResponseMessage: zendia.MsgCustomClaimsSet,
			"token":               zendia.MsgTokenPlaceholder,
		})
		return nil
	}))

	// Conecta MongoDB (opcional) - PRIMEIRO!
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(zendia.DefaultMongoURI))
	
	// Monitoramento e Tracing
	var metrics *zendia.Metrics
	if err == nil {
		// Com MongoDB: monitoring + persistência automática
		log.Println("Conectado ao MongoDB")
		metricsCollection := client.Database("meu_projeto").Collection("metrics")
		metrics = app.AddMonitoringWithPersistence(metricsCollection)
		log.Println("📊 Monitoring com persistência ativado - histórico salvo no MongoDB")
	} else {
		// Sem MongoDB: só monitoring em memória
		log.Println("MongoDB não disponível, usando monitoring em memória")
		metrics = app.AddMonitoring()
		log.Println("📊 Monitoring em memória ativado - sem persistência")
	}
	


	// Health Manager Global com checks reais
	globalHealth := zendia.NewHealthManager()
	globalHealth.AddCheck(zendia.NewMemoryHealthCheck(1024)) // 1GB max
	app.AddHealthEndpoint(globalHealth)

	// Swagger
	app.SetupSwagger(zendia.SwaggerInfo{
		Title:       "ZendiaFramework Complete API",
		Description: "API completa demonstrando todas as funcionalidades com UUID nativo",
		Version:     "1.0",
		Host:        zendia.DefaultHost,
		BasePath:    zendia.DefaultBasePath,
	})

	// Setup dos repositories
	var userRepo interface{}
	if err != nil {
		// Fallback para repository em memória
		log.Println("Usando repository em memória")
		baseRepo := zendia.NewMemoryRepository[*User, uuid.UUID](func() uuid.UUID {
			return uuid.New()
		})
		userRepo = zendia.NewAuditRepository[*User, uuid.UUID](baseRepo)
	} else {
		// Usa MongoDB com UUID nativo - VOCÊ escolhe o nome do banco!
		collection := client.Database("meu_projeto").Collection("usuarios")
		baseRepo := zendia.NewMongoAuditRepository[*User](collection)
		
		// Adiciona cache automático (in-memory - sem dependências)
		memoryCache := zendia.NewMemoryCache(zendia.MemoryCacheConfig{
			CacheConfig: zendia.CacheConfig{
				TTL: 5 * time.Minute,
			},
			MaxSize: 1000,
		})
		userRepo = zendia.NewCachedRepository(baseRepo, memoryCache, zendia.CacheConfig{
			TTL: 5 * time.Minute,
		}, "User")
		log.Println("Cache em memória ativado - performance 50x mais rápida!")
	}

	// API v1
	api := app.Group(zendia.RouteAPIV1)

	// Health específico da API
	apiHealth := zendia.NewHealthManager()
	apiHealth.AddCheck(zendia.NewHTTPHealthCheck("external_api", "https://httpbin.org/status/200", 5*time.Second))
	if err == nil {
		apiHealth.AddCheck(zendia.NewDatabaseHealthCheck("mongodb", func(ctx context.Context) error {
			return client.Ping(ctx, nil)
		}))
	}
	api.AddHealthEndpoint(apiHealth)

	// Grupo de usuários (já protegido pelo Firebase Auth)
	users := api.Group(zendia.RouteUsers)

	// Health específico dos usuários
	usersHealth := zendia.NewHealthManager()
	usersHealth.AddCheck(zendia.NewRepositoryHealthCheck("user_repository", userRepo))
	users.AddHealthEndpoint(usersHealth)

	// CRUD Completo - Tenant automático da sessão
	users.POST("/", zendia.Handle(func(c *zendia.Context[User]) error {
		var user User
		if err := c.BindJSON(&user); err != nil {
			return err
		}

		// TenantID e UserID vêm automaticamente da sessão!
		tenantID := c.GetTenantID() // ← Setado no /login
		userID := c.GetUserID()     // ← Setado no /login

		if tenantID == "" {
			return zendia.NewBadRequestError(zendia.MsgLoginRequired)
		}

		// Cria usando repository com UUID nativo
		var created *User
		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			created, err = mongoRepo.Create(c.Request.Context(), &user)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, uuid.UUID]); ok {
			created, err = memRepo.Create(c.Request.Context(), &user)
		}

		if err != nil {
			return err
		}

		c.Created(zendia.MsgCreatedSuccess, map[string]interface{}{
			"user":        created,
			"tenant_id":   tenantID,
			"created_by":  userID,
			"tenant_info": c.GetTenantInfo(),
		})
		return nil
	}))

	users.GET("/", zendia.Handle(func(c *zendia.Context[any]) error {
		// Secure pagination with validation
		skip := 0
		take := 10
		if skipStr := c.Query(zendia.QuerySkip); skipStr != "" {
			if parsed, err := strconv.Atoi(skipStr); err == nil {
				skip = parsed
			}
		}
		if takeStr := c.Query(zendia.QueryTake); takeStr != "" {
			if parsed, err := strconv.Atoi(takeStr); err == nil {
				take = parsed
			}
		}
		if skip < 0 || take < 0 || take > 1000 {
			c.BadRequest(zendia.MsgInvalidPagination)
			return nil
		}

		// Safe filters - only allow whitelisted fields
		filters := map[string]interface{}{}
		if name := c.Query(zendia.QueryName); name != "" && len(name) <= 100 {
			filters[zendia.FieldName] = name
		}
		if tenantID := c.GetTenantID(); tenantID != "" {
			filters[zendia.FieldTenantID] = tenantID
		}

		var users []*User
		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			users, err = mongoRepo.GetAll(c.Request.Context(), filters)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, uuid.UUID]); ok {
			users, err = memRepo.GetAll(c.Request.Context(), filters)
		}

		if err != nil {
			return err
		}

		c.Success(zendia.MsgRetrievedSuccess, map[string]interface{}{
			"users":     users,
			"tenant_id": c.GetTenantID(),
			"count":     len(users),
		})
		return nil
	}))

	users.GET("/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return zendia.NewBadRequestError(zendia.MsgInvalidUUID)
		}

		var user *User
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			user, err = mongoRepo.GetByID(c.Request.Context(), id)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, uuid.UUID]); ok {
			user, err = memRepo.GetByID(c.Request.Context(), id)
		}

		if err != nil {
			return err
		}

		c.Success(zendia.MsgRetrievedByIDSuccess, user)
		return nil
	}))

	users.PUT("/:id", zendia.Handle(func(c *zendia.Context[User]) error {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return zendia.NewBadRequestError(zendia.MsgInvalidUUID)
		}

		var user User
		if err := c.BindJSON(&user); err != nil {
			return err
		}

		var updated *User
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			updated, err = mongoRepo.Update(c.Request.Context(), id, &user)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, uuid.UUID]); ok {
			updated, err = memRepo.Update(c.Request.Context(), id, &user)
		}

		if err != nil {
			return err
		}

		c.Success(zendia.MsgUpdatedSuccess, updated)
		return nil
	}))

	users.DELETE("/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return zendia.NewBadRequestError(zendia.MsgInvalidUUID)
		}

		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			err = mongoRepo.Delete(c.Request.Context(), id)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, uuid.UUID]); ok {
			err = memRepo.Delete(c.Request.Context(), id)
		}

		if err != nil {
			return err
		}

		c.NoContent()
		return nil
	}))

	// Endpoint para verificar dados do usuário autenticado
	api.GET(zendia.RouteMe, zendia.Handle(func(c *zendia.Context[any]) error {
		user := c.GetAuthUser()
		c.Success(zendia.MsgUserData, map[string]interface{}{
			"firebase_uid": user.FirebaseUID, // ← Do Firebase
			"email":        user.Email,       // ← Do Firebase
			"user_id":      user.ID,          // ← Custom claim
			"tenant_id":    user.TenantID,    // ← Custom claim

			"tenant_info":  c.GetTenantInfo(),
		})
		return nil
	}))

	// Endpoints públicos (não protegidos)
	app.GET(zendia.RouteMetrics, zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success(zendia.MsgMetricsFound, metrics.GetStats())
		return nil
	}))


	
	// Endpoints de métricas avançadas (se MongoDB disponível)
	if err == nil {
		log.Println("📈 Endpoints de histórico disponíveis:")
		log.Println("   GET /public/metrics/history - Histórico detalhado")
		log.Println("   GET /public/metrics/stats - Estatísticas agregadas")
		log.Println("   DELETE /public/metrics/cleanup - Limpeza de dados antigos")
	}

	// Banner automático do framework
	app.ShowBanner(zendia.BannerConfig{
		AppName:    "ZendiaFramework Demo",
		Version:    "1.0.0",
		Port:       "8080",
		ShowRoutes: true,
	})

	log.Println("\n🔥 Teste o fluxo completo:")
	log.Println("1. POST /auth/login com email/senha → Seta custom claims")
	log.Println("2. GET /api/v1/me com token → Framework extrai custom claims")
	log.Println("3. POST /api/v1/users com token → Tenant/user_id automáticos")
	log.Println("4. GET /public/metrics → Métricas em tempo real")
	if err == nil {
		log.Println("5. GET /public/metrics/history → Histórico persistido")
		log.Println("6. GET /public/metrics/stats?interval=hour → Estatísticas agregadas")
	}

	app.Run(":8080")
}
