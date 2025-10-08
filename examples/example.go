package main

import (
	"context"
	"log"
	"time"

	"github.com/azzidev/zendiaframework"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User entidade completa com auditoria e tenant
type User struct {
	ID        string    `bson:"_id" json:"id"`
	Name      string    `bson:"name" json:"name" validate:"required,min=2,max=50"`
	Email     string    `bson:"email" json:"email" validate:"required,email"`
	Age       int       `bson:"age" json:"age" validate:"gte=0,lte=120"`
	TenantID  string    `bson:"tenant_id" json:"tenant_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
	CreatedBy string    `bson:"created_by" json:"created_by"`
	UpdatedBy string    `bson:"updated_by" json:"updated_by"`
}

// Implementa MongoAuditableEntity
func (u *User) GetID() string { return u.ID }
func (u *User) SetID(id string) { u.ID = id }
func (u *User) SetCreatedAt(t time.Time) { u.CreatedAt = t }
func (u *User) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }
func (u *User) SetCreatedBy(s string)    { u.CreatedBy = s }
func (u *User) SetUpdatedBy(s string)    { u.UpdatedBy = s }
func (u *User) SetTenantID(s string)     { u.TenantID = s }

func main() {
	app := zendia.New()

	// Middlewares
	app.Use(zendia.Logger())
	app.Use(zendia.CORS())
	app.Use(zendia.Compression())

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
		Description: "API completa demonstrando todas as funcionalidades",
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
		baseRepo := zendia.NewMemoryRepository[*User, string](func() string {
			return uuid.New().String()
		})
		userRepo = zendia.NewAuditRepository[*User, string](baseRepo)
	} else {
		// Usa MongoDB
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

	// Grupo de usuários com auth
	users := api.Group("/users", zendia.Auth(func(token string) bool {
		return token == "valid-token" // Validação simples
	}))

	// Health específico dos usuários
	usersHealth := zendia.NewHealthManager()
	usersHealth.AddCheck(zendia.NewRepositoryHealthCheck("user_repository", userRepo))
	users.AddHealthEndpoint(usersHealth)

	// CRUD Completo
	users.POST("/", zendia.Handle(func(c *zendia.Context[User]) error {
		var user User
		if err := c.BindJSON(&user); err != nil {
			return err
		}

		// Cria usando qualquer tipo de repository
		var created *User
		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			created, err = mongoRepo.Create(c.Request.Context(), &user)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, string]); ok {
			created, err = memRepo.Create(c.Request.Context(), &user)
		}

		if err != nil {
			return err
		}

		c.Created(map[string]interface{}{
			"user": created,
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
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, string]); ok {
			users, err = memRepo.GetAll(c.Request.Context(), filters)
		}

		if err != nil {
			return err
		}

		c.Success(map[string]interface{}{
			"users": users,
			"tenant_id": c.GetTenantID(),
			"count": len(users),
		})
		return nil
	}))

	users.GET("/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		id := c.Param("id")

		var user *User
		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			user, err = mongoRepo.GetByID(c.Request.Context(), id)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, string]); ok {
			user, err = memRepo.GetByID(c.Request.Context(), id)
		}

		if err != nil {
			return err
		}

		c.Success(user)
		return nil
	}))

	users.PUT("/:id", zendia.Handle(func(c *zendia.Context[User]) error {
		id := c.Param("id")
		var user User
		if err := c.BindJSON(&user); err != nil {
			return err
		}

		var updated *User
		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			updated, err = mongoRepo.Update(c.Request.Context(), id, &user)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, string]); ok {
			updated, err = memRepo.Update(c.Request.Context(), id, &user)
		}

		if err != nil {
			return err
		}

		c.Success(updated)
		return nil
	}))

	users.DELETE("/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		id := c.Param("id")

		var err error
		if mongoRepo, ok := userRepo.(*zendia.MongoAuditRepository[*User]); ok {
			err = mongoRepo.Delete(c.Request.Context(), id)
		} else if memRepo, ok := userRepo.(*zendia.AuditRepository[*User, string]); ok {
			err = memRepo.Delete(c.Request.Context(), id)
		}

		if err != nil {
			return err
		}

		c.NoContent()
		return nil
	}))

	// Endpoints de sistema
	app.GET("/metrics", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success(metrics.GetStats())
		return nil
	}))

	app.GET("/traces", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success(tracer.GetSpans())
		return nil
	}))

	app.GET("/tenant-info", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success(c.GetTenantInfo())
		return nil
	}))

	log.Println("🚀 ZendiaFramework Demo rodando em :8080")
	log.Println("")
	log.Println("📋 Headers necessários:")
	log.Println("  X-Tenant-ID: ID do tenant")
	log.Println("  X-User-ID: ID do usuário")
	log.Println("  Authorization: Bearer valid-token (para /users)")
	log.Println("")
	log.Println("🔗 Endpoints disponíveis:")
	log.Println("  GET  /health - Health check global")
	log.Println("  GET  /api/v1/health - Health check da API")
	log.Println("  GET  /api/v1/users/health - Health check dos usuários")
	log.Println("  GET  /metrics - Métricas da aplicação")
	log.Println("  GET  /traces - Spans de tracing")
	log.Println("  GET  /tenant-info - Informações do tenant")
	log.Println("  GET  /swagger/index.html - Documentação Swagger")
	log.Println("")
	log.Println("👥 CRUD Usuários (requer auth):")
	log.Println("  POST /api/v1/users - Criar usuário")
	log.Println("  GET  /api/v1/users - Listar usuários")
	log.Println("  GET  /api/v1/users/:id - Buscar usuário")
	log.Println("  PUT  /api/v1/users/:id - Atualizar usuário")
	log.Println("  DELETE /api/v1/users/:id - Deletar usuário")
	log.Println("")
	log.Println("💡 Exemplo de teste:")
	log.Println("curl -H 'X-Tenant-ID: demo' -H 'X-User-ID: user1' http://localhost:8080/tenant-info")

	app.Run(":8080")
}