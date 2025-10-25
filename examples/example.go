package main

import (
	"context"
	"log"
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
	ID        uuid.UUID `bson:"_id" json:"id"`
	Name      string    `bson:"name" json:"name" validate:"required,min=2,max=50"`
	Email     string    `bson:"email" json:"email" validate:"required,email"`
	Age       int       `bson:"age" json:"age" validate:"gte=0,lte=120"`
	TenantID  uuid.UUID `bson:"tenant_id" json:"tenant_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
	CreatedBy string    `bson:"created_by" json:"created_by"`
	UpdatedBy string    `bson:"updated_by" json:"updated_by"`
}

// Implementa MongoAuditableEntity com UUID nativo
func (u *User) GetID() uuid.UUID         { return u.ID }
func (u *User) SetID(id uuid.UUID)       { u.ID = id }
func (u *User) SetCreatedAt(t time.Time) { u.CreatedAt = t }
func (u *User) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }
func (u *User) SetCreatedBy(s string)    { u.CreatedBy = s }
func (u *User) SetUpdatedBy(s string)    { u.UpdatedBy = s }
func (u *User) SetTenantID(s string) {
	if s != "" {
		u.TenantID = uuid.MustParse(s)
	}
}

func main() {
	// Inicializa Firebase
	ctx := context.Background()
	// Use suas credenciais Firebase
	opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
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
		PublicRoutes:   []string{"/public", "/docs", "/auth"},
	})

	// Rota de login PÚBLICA: email/senha → Firebase token + custom claims
	app.POST("/auth/login", zendia.Handle(func(c *zendia.Context[any]) error {
		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required"`
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

		// 4. Seta custom claims (PARA SEMPRE)
		// claims := map[string]interface{}{
		//     "tenant_id": userFromDB.TenantID,
		//     "user_id":   userFromDB.ID,
		//     "role":      userFromDB.Role,
		// }
		// err = firebaseAuth.SetCustomUserClaims(ctx, decodedToken.UID, claims)

		c.Success("Login realizado", map[string]interface{}{
			"message": "Custom claims setados - token funciona para sempre",
			"token":   "firebase-token-with-custom-claims",
		})
		return nil
	}))

	// Monitoramento e Tracing
	metrics := app.AddMonitoring()
	tracer := zendia.NewSimpleTracer()
	app.Use(zendia.Tracing(tracer))

	// Health Manager Global com checks reais
	globalHealth := zendia.NewHealthManager()
	globalHealth.AddCheck(zendia.NewMemoryHealthCheck(1024)) // 1GB max
	app.AddHealthEndpoint(globalHealth)

	// Swagger
	app.SetupSwagger(zendia.SwaggerInfo{
		Title:       "ZendiaFramework Complete API",
		Description: "API completa demonstrando todas as funcionalidades com UUID nativo",
		Version:     "1.0",
		Host:        "localhost:8080",
		BasePath:    "/api/v1",
	})

	// Conecta MongoDB (opcional)
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	var userRepo interface{}
	if err != nil {
		// Fallback para repository em memória
		log.Println("MongoDB não disponível, usando repository em memória")
		baseRepo := zendia.NewMemoryRepository[*User, uuid.UUID](func() uuid.UUID {
			return uuid.New()
		})
		userRepo = zendia.NewAuditRepository[*User, uuid.UUID](baseRepo)
	} else {
		// Usa MongoDB com UUID nativo
		log.Println("Conectado ao MongoDB")
		collection := client.Database("zendia_demo").Collection("users")
		userRepo = zendia.NewMongoAuditRepository[*User](collection)
	}

	// API v1
	api := app.Group("/api/v1")

	// Health específico da API
	apiHealth := zendia.NewHealthManager()
	apiHealth.AddCheck(zendia.NewHTTPHealthCheck("external_api", "https://httpbin.org/status/200", 5*time.Second))
	if client != nil {
		apiHealth.AddCheck(zendia.NewDatabaseHealthCheck("mongodb", func(ctx context.Context) error {
			return client.Ping(ctx, nil)
		}))
	}
	api.AddHealthEndpoint(apiHealth)

	// Grupo de usuários (já protegido pelo Firebase Auth)
	users := api.Group("/users")

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
			return zendia.NewBadRequestError("Faça login primeiro para setar o tenant")
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

		c.Created("Criado com sucesso.", map[string]interface{}{
			"user":        created,
			"tenant_id":   tenantID,
			"created_by":  userID,
			"tenant_info": c.GetTenantInfo(),
		})
		return nil
	}))

	users.GET("/", zendia.Handle(func(c *zendia.Context[any]) error {
		filters := map[string]interface{}{}
		if name := c.Query("name"); name != "" {
			filters["name"] = name
		}
		if tenantID := c.GetTenantID(); tenantID != "" {
			filters["tenant_id"] = tenantID
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

		c.Success("Capturado com sucesso.", map[string]interface{}{
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
			return zendia.NewBadRequestError("Invalid UUID format")
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

		c.Success("Capturado com sucesso usando ID.", user)
		return nil
	}))

	users.PUT("/:id", zendia.Handle(func(c *zendia.Context[User]) error {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return zendia.NewBadRequestError("Invalid UUID format")
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

		c.Success("Atualizado com sucesso.", updated)
		return nil
	}))

	users.DELETE("/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return zendia.NewBadRequestError("Invalid UUID format")
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
	api.GET("/me", zendia.Handle(func(c *zendia.Context[any]) error {
		user := c.GetAuthUser()
		c.Success("Dados do usuário", map[string]interface{}{
			"firebase_uid": user.FirebaseUID, // ← Do Firebase
			"email":        user.Email,       // ← Do Firebase
			"user_id":      user.ID,          // ← Custom claim
			"tenant_id":    user.TenantID,    // ← Custom claim
			"role":         user.Role,        // ← Custom claim
			"tenant_info":  c.GetTenantInfo(),
		})
		return nil
	}))

	// Endpoints públicos (não protegidos)
	app.GET("/public/metrics", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success("Metricas encontradas.", metrics.GetStats())
		return nil
	}))

	app.GET("/public/traces", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success("Traces encontradas.", tracer.GetSpans())
		return nil
	}))

	// Banner automático do framework
	app.ShowBanner(zendia.BannerConfig{
		AppName:    "ZendiaFramework Demo",
		Version:    "1.0.0",
		Port:       "8080",
		ShowRoutes: true,
	})

	log.Println("\n🔥 Teste o fluxo com custom claims:")
	log.Println("1. POST /auth/login com email/senha → Seta custom claims")
	log.Println("2. GET /api/v1/me com token → Framework extrai custom claims")
	log.Println("3. POST /api/v1/users com token → Tenant/user_id automáticos")

	app.Run(":8080")
}
