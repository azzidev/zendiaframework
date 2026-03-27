package main

import (
	"context"
	"log"
	"time"

	zendia "github.com/azzidev/zendiaframework"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User entidade de exemplo
type User struct {
	ID       uuid.UUID        `bson:"_id" json:"id"`
	Name     string           `json:"name" validate:"required,min=2"`
	Email    string           `json:"email" validate:"required,email"`
	Age      int              `json:"age" validate:"gte=0,lte=150"`
	TenantID uuid.UUID        `bson:"tenant_id" json:"tenant_id"`
	Active   bool             `bson:"active" json:"active"`
	Created  zendia.AuditInfo `bson:"created" json:"created"`
	Updated  zendia.AuditInfo `bson:"updated" json:"updated"`
	Deleted  zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"`
}

func (u *User) GetID() uuid.UUID                    { return u.ID }
func (u *User) SetID(id uuid.UUID)                  { u.ID = id }
func (u *User) SetTenantID(s string)                { u.TenantID = uuid.MustParse(s) }
func (u *User) SetCreated(info zendia.AuditInfo)     { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo)     { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo)     { u.Deleted = info }
func (u *User) SetActive(active bool)                { u.Active = active }

func main() {
	app := zendia.New()
	app.Use(zendia.Logger())
	app.Use(zendia.CORS("*"))

	// MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}
	db := client.Database("zendia_example")

	// Repository simples (sem auditoria, sem histórico)
	_ = zendia.NewRepository[*User](db.Collection("users_simple"))

	// Repository com auditoria (tenant injection, created/updated/deleted automáticos)
	_ = zendia.NewRepository[*User](db.Collection("users_audit"), zendia.WithAudit())

	// Repository com auditoria + histórico de mudanças
	userRepo := zendia.NewRepository[*User](
		db.Collection("users"),
		zendia.WithAudit(),
		zendia.WithHistory(db.Collection("history"), "User"),
	)

	// Repository com cache
	cache := zendia.NewMemoryCache(zendia.MemoryCacheConfig{
		CacheConfig: zendia.CacheConfig{TTL: 10 * time.Minute},
		MaxSize:     10000,
	})
	cachedRepo := zendia.NewCachedRepository(userRepo, cache, zendia.CacheConfig{
		TTL: 10 * time.Minute,
	}, "User")
	_ = cachedRepo

	// Monitoring
	metrics := app.AddMonitoring()

	// Health checks
	globalHealth := zendia.NewHealthManager()
	globalHealth.AddCheck(zendia.NewMemoryHealthCheck(2048))
	globalHealth.AddCheck(zendia.NewDatabaseHealthCheck("mongodb", func(ctx context.Context) error {
		return client.Ping(ctx, nil)
	}))
	app.AddHealthEndpoint(globalHealth)

	// Metrics endpoint
	app.GET("/public/metrics", zendia.Handle(func(c *zendia.Context[any]) error {
		c.Success("Métricas capturadas.", metrics.GetStats())
		return nil
	}))

	// CRUD routes
	api := app.Group("/api/v1")

	// Create user
	api.POST("/users", zendia.Handle(func(c *zendia.Context[User]) error {
		var user User
		if err := c.BindJSON(&user); err != nil {
			return err
		}

		created, err := userRepo.Create(c.Request.Context(), &user)
		if err != nil {
			return err
		}

		c.Created("Usuário criado com sucesso", created)
		return nil
	}))

	// Get user
	api.GET("/users/:id", zendia.Handle(func(c *zendia.Context[any]) error {
		id, err := uuid.Parse(c.Context.Param("id"))
		if err != nil {
			return zendia.NewBadRequestError("ID inválido")
		}

		user, err := userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			return err
		}

		c.Success("Usuário encontrado", user)
		return nil
	}))

	// List users
	api.GET("/users", zendia.Handle(func(c *zendia.Context[any]) error {
		users, err := userRepo.GetAll(c.Request.Context(), map[string]interface{}{})
		if err != nil {
			return err
		}

		c.Success("Usuários encontrados", users)
		return nil
	}))

	// Get history
	api.GET("/users/:id/history", zendia.Handle(func(c *zendia.Context[any]) error {
		id, err := uuid.Parse(c.Context.Param("id"))
		if err != nil {
			return zendia.NewBadRequestError("ID inválido")
		}

		history, err := userRepo.GetHistory(c.Request.Context(), id)
		if err != nil {
			return err
		}

		c.Success("Histórico encontrado", history)
		return nil
	}))

	// Swagger
	app.SetupSwagger(zendia.SwaggerInfo{
		Title:       "Zendia Example API",
		Description: "API de exemplo usando ZendiaFramework",
		Version:     "1.0.0",
	})

	// Banner
	app.ShowBanner(zendia.BannerConfig{
		AppName:    "Zendia Example",
		Version:    "1.0.0",
		Port:       "8080",
		ShowRoutes: true,
	})

	app.Run(":8080")
}
