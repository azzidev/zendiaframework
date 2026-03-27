<div align="center">
  <h1>
    <svg width="1em" height="1em" viewBox="0 0 100 100" style="display: inline-block; vertical-align: middle; margin-right: 0.2em;">
      <defs>
        <linearGradient id="zGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" style="stop-color:#00ADD8;stop-opacity:1" />
          <stop offset="100%" style="stop-color:#0066CC;stop-opacity:1" />
        </linearGradient>
      </defs>
      <path d="M15 25 L75 25 L35 65 L85 65 L85 75 L25 75 L65 35 L15 35 Z" fill="url(#zGradient)" stroke="#003366" stroke-width="2"/>
    </svg>
    ZendiaFramework
  </h1>
  <p><strong>Framework Go modular e poderoso para APIs RESTful</strong></p>

  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Build Status](https://img.shields.io/badge/Build-Passing-success?style=for-the-badge)]()

<p>Construído sobre o Gin com foco em <strong>simplicidade</strong>, <strong>performance</strong> e <strong>flexibilidade</strong></p>
</div>

---

## ✨ Características Principais

### 🎯 **Core Features**

- 🛣️ **Roteamento Inteligente** - Sistema flexível com grupos e middlewares
- 🔒 **Multi-Tenant** - Contexto automático de tenant/usuário em todas as requisições
- 📊 **Monitoramento Built-in** - Métricas, tracing e health checks nativos
- 🗄️ **Repository Unificado** - MongoDB com options (`WithAudit`, `WithHistory`)
- ⚡ **Generics** - Type-safe com suporte completo a generics do Go

### 🛡️ **Segurança & Qualidade**

- 🔐 **Autenticação** - Sistema flexível de auth com tokens
- ✅ **Validação Robusta** - Validação automática com mensagens em português
- 🔍 **Tenant Isolation** - Filtros automáticos por tenant em todas as queries
- 🚨 **Error Handling** - Tratamento padronizado e consistente de erros
- 📝 **Auditoria & Histórico** - Tracking automático + histórico de mudanças
- 🕒 **Change Tracking** - Registra apenas campos alterados com metadados

### 🔧 **DevEx & Produção**

- 📚 **Swagger Automático** - Documentação gerada automaticamente
- 🏥 **Health Checks Reais** - Monitoramento com dados reais do sistema
- 📈 **Observabilidade** - Métricas detalhadas e health checks
- 🧪 **Testável** - Arquitetura que facilita testes unitários

---

## 🚀 Quick Start

### Instalação

```bash
go get github.com/azzidev/zendiaframework
go get firebase.google.com/go/v4
```

### Hello World com Firebase Auth

```go
package main

import (
    "context"
    "log"
    "github.com/azzidev/zendiaframework"
    firebase "firebase.google.com/go/v4"
    "google.golang.org/api/option"
)

func main() {
    // Inicializa Firebase
    opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
    firebaseApp, err := firebase.NewApp(context.Background(), nil, opt)
    if err != nil {
        log.Fatal("Firebase init failed:", err)
    }
    firebaseAuth, err := firebaseApp.Auth(context.Background())
    if err != nil {
        log.Fatal("Firebase Auth init failed:", err)
    }
    
    app := zendia.New()
    
    // Setup Firebase Auth - extrai custom claims automaticamente
    app.SetupFirebaseAuth(zendia.FirebaseAuthConfig{
        FirebaseClient: firebaseAuth,
        PublicRoutes:   []string{"/public", "/auth"},
    })
    
    // Login PÚBLICO: email/senha → Firebase token + custom claims
    app.POST("/auth/login", zendia.Handle(func(c *zendia.Context[any]) error {
        var req struct {
            Email    string `json:"email" validate:"required,email"`
            Password string `json:"password" validate:"required"`
        }
        if err := c.Context.ShouldBindJSON(&req); err != nil {
            return err
        }
        
        // 1. Autentica no Firebase
        token, err := authenticateFirebase(req.Email, req.Password)
        if err != nil {
            c.Unauthorized("Credenciais inválidas")
            return nil
        }
        
        // 2. Decodifica token para pegar Firebase UID
        decodedToken, err := firebaseAuth.VerifyIDToken(c.Request.Context(), token)
        if err != nil {
            c.Unauthorized("Token inválido")
            return nil
        }
        
        // 3. Busca usuário no SEU banco
        // userFromDB := myRepo.FindByEmail(req.Email)
        
        // 4. Seta custom claims (PARA SEMPRE) - USE AS CONSTANTES!
        claims := map[string]interface{}{
            zendia.ClaimTenantID: "company-123",  // ← Do seu banco
            zendia.ClaimUserUUID: "user-456",     // ← ID do seu sistema
            zendia.ClaimUserName: "John Doe",     // ← Nome do usuário
            zendia.ClaimRole:     "admin",        // ← Role do seu sistema
        }
        err = firebaseAuth.SetCustomUserClaims(c.Request.Context(), decodedToken.UID, claims)
        if err != nil {
            c.InternalError("Falha ao configurar sessão")
            return nil
        }
        
        c.Success("Login realizado", map[string]interface{}{
            "token": token,
        })
        return nil
    }))
  
    // Rota protegida - framework extrai custom claims automaticamente
    api := app.Group("/api/v1")
    api.GET("/hello", zendia.Handle(func(c *zendia.Context[any]) error {
        user := c.GetAuthUser()
        
        c.Success("Hello from ZendiaFramework! 🎉", map[string]interface{}{
            "firebase_uid": user.FirebaseUID, // ← Do Firebase
            "email":        user.Email,       // ← Do Firebase
            "user_id":      user.ID,          // ← Custom claim
            "tenant":       user.TenantID,    // ← Custom claim
            "role":         user.Role,        // ← Custom claim
        })
        return nil
    }))
    
    // Rota pública
    app.GET("/public/status", zendia.Handle(func(c *zendia.Context[any]) error {
        c.Success("Status OK", map[string]string{"status": "ok"})
        return nil
    }))
  
    app.Run(":8080")
}
```

### Teste com Custom Claims

```bash
# 1. Login PÚBLICO (email/senha → Firebase token + custom claims)
curl -X POST -H "Content-Type: application/json" \
     -d '{"email":"user@company.com","password":"123456"}' \
     http://localhost:8080/auth/login

# 2. Rota protegida (framework extrai custom claims automaticamente)
curl -H "Authorization: Bearer <firebase-token>" \
     http://localhost:8080/api/v1/hello

# 3. Rota pública (sem token)
curl http://localhost:8080/public/status
```

---

## 🏗️ Arquitetura

### Multi-Tenant por Padrão

```go
// Contexto de tenant automático após login
func createUser(c *zendia.Context[User]) error {
    // TenantID vem da sessão (setado no login)
    tenantID := c.GetTenantID()  // ← Da sessão
    userID := c.GetUserID()      // ← Da sessão
    
    if tenantID == "" {
        return zendia.NewBadRequestError("Faça login primeiro")
    }
  
    var user User
    c.BindJSON(&user) // Validação automática
  
    // Auditoria automática (CreatedBy, CreatedAt, TenantID)
    created, err := userRepo.Create(c.Request.Context(), &user)
  
    c.Created("Usuário criado", created)
    return nil
}
```

### Repository Unificado com Options

```go
import "github.com/google/uuid"

type User struct {
    ID        uuid.UUID `bson:"_id" json:"id"`
    Name      string    `json:"name" validate:"required,min=2"`
    Email     string    `json:"email" validate:"required,email"`
    TenantID  uuid.UUID `bson:"tenant_id" json:"tenant_id"`
    Active    bool      `bson:"active" json:"active"`
    Created   zendia.AuditInfo `bson:"created" json:"created"`
    Updated   zendia.AuditInfo `bson:"updated" json:"updated"`
    Deleted   zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"`
}

// Implementa interface para auditoria automática
func (u *User) GetID() uuid.UUID              { return u.ID }
func (u *User) SetID(id uuid.UUID)            { u.ID = id }
func (u *User) SetCreated(info zendia.AuditInfo) { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo) { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo) { u.Deleted = info }
func (u *User) SetTenantID(s string)          { u.TenantID = uuid.MustParse(s) }
func (u *User) SetActive(active bool)         { u.Active = active }

// Simples - sem auditoria, sem histórico
repo := zendia.NewRepository[*User](collection)

// Com auditoria (tenant injection, created/updated/deleted automáticos)
repo := zendia.NewRepository[*User](collection, zendia.WithAudit())

// Com auditoria + histórico de mudanças
repo := zendia.NewRepository[*User](collection,
    zendia.WithAudit(),
    zendia.WithHistory(historyCollection, "User"),
)

// Com cache automático (in-memory - sem dependências)
memoryCache := zendia.NewMemoryCache(zendia.MemoryCacheConfig{
    CacheConfig: zendia.CacheConfig{
        TTL: 10 * time.Minute,
    },
    MaxSize: 10000,
})
cachedRepo := zendia.NewCachedRepository(repo, memoryCache, zendia.CacheConfig{
    TTL: 10 * time.Minute,
}, "User")
```

> **Soft Delete**: O framework usa `active: true/false` para soft delete. Todos os métodos de leitura filtram automaticamente por `active: true`. O campo `deleted` é preenchido com informações de auditoria quando `WithAudit()` está habilitado, mas **não é usado como filtro**.
```

### 🔄 QueryOptions

```go
// Ordenação, limite, skip e projeção
queryOpts := &zendia.QueryOptions{
    Sort:       map[string]interface{}{"created.set_at": -1},
    Limit:      10,
    Skip:       20,
    Projection: map[string]interface{}{"name": 1, "email": 1},
}

// Aplicar em qualquer método de busca
users, err := repo.GetAll(ctx, filters, queryOpts)
users, err := repo.GetAllSkipTake(ctx, filters, 0, 10, queryOpts)
users, err := repo.List(ctx, filters, queryOpts)

// Múltiplos campos de ordenação
queryOpts := &zendia.QueryOptions{
    Sort: map[string]interface{}{
        "priority": -1,        // Prioridade decrescente primeiro
        "created.set_at": 1,  // Depois por data crescente
    },
}

// Casos de uso comuns
// Notificações mais recentes primeiro
notifications, err := notificationRepo.GetAllSkipTake(ctx, filters, 0, 20, &zendia.QueryOptions{
    Sort: map[string]interface{}{"created.set_at": -1},
})

// Usuários por nome alfabético
users, err := userRepo.GetAll(ctx, filters, &zendia.QueryOptions{
    Sort: map[string]interface{}{"name": 1},
})
```

---

## 🛠️ Funcionalidades Avançadas

### 🚀 Cache Layer Automático

```go
// Opção 1: Cache em memória (sem dependências)
memoryCache := zendia.NewMemoryCache(zendia.MemoryCacheConfig{
    CacheConfig: zendia.CacheConfig{
        TTL: 10 * time.Minute, // Expira em 10min
    },
    MaxSize:   10000,           // Máximo 10k itens
    MaxMemory: 5 * 1024 * 1024, // Máximo 5MB
})

// Opção 2: Cache Redis (para produção)
// import "github.com/redis/go-redis/v9"
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
redisCache := zendia.NewRedisCache(zendia.RedisCacheConfig{
    Client: redisClient,
    TTL:    10 * time.Minute,
})

// Mesmo uso para ambos!
cachedRepo := zendia.NewCachedRepository(baseRepo, memoryCache, zendia.CacheConfig{
    TTL: 10 * time.Minute,
}, "User")

// Performance automática:
user, err := cachedRepo.GetByID(ctx, userID)
// Primeira vez: MongoDB (50ms)
// Próximas vezes: Cache (0.1ms) 🚀

// Invalidação automática:
user.Name = "Novo Nome"
cachedRepo.Update(ctx, userID, user)  // ← Remove do cache automaticamente!
```

### 📊 Monitoramento e Observabilidade Completa

```go
app := zendia.New()

// Opção 1: Monitoring simples (só memória)
metrics := app.AddMonitoring()

// Opção 2: Monitoring com persistência MongoDB (RECOMENDADO)
client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
metricsCollection := client.Database("myapp").Collection("metrics")
metrics := app.AddMonitoringWithPersistence(metricsCollection)
// ✅ Salva métricas a cada 1 minuto automaticamente
// ✅ Histórico completo com TTL de 30 dias
// ✅ Endpoints de consulta automáticos

// Health checks granulares
globalHealth := zendia.NewHealthManager()
globalHealth.AddCheck(zendia.NewDatabaseHealthCheck("main_db", dbPing))
app.AddHealthEndpoint(globalHealth) // GET /health

// Repository com histórico automático
projectRepo := zendia.NewRepository[*Project](
    db.Collection("projects"),
    zendia.WithAudit(),
    zendia.WithHistory(db.Collection("history"), "Project"),
)

// Histórico automático em updates
project, err := projectRepo.Update(ctx, id, updatedProject)
// Registra automaticamente apenas os campos que mudaram!

// Consultar histórico
history, err := projectRepo.GetHistory(ctx, projectID)
// Retorna: [{"Name": {"before": "Old", "after": "New"}}]
```

#### 📈 Endpoints de Métricas Disponíveis

```bash
# Métricas em tempo real
GET /public/metrics

# Histórico detalhado (últimas 24h)
GET /public/metrics/history

# Histórico por período
GET /public/metrics/history?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z

# Estatísticas agregadas (por hora/dia/mês)
GET /public/metrics/stats?interval=day&from=2024-01-01T00:00:00Z

# Limpeza de dados antigos
DELETE /public/metrics/cleanup?days=30
```

#### 📉 Exemplo de Resposta com Persistência

```json
{
  "success": true,
  "message": "Métricas encontradas",
  "data": {
    "uptime": "2h30m15s",
    "active_requests": 5,
    "total_requests": 1250,
    "total_errors": 23,
    "error_rate": 1.84,
    "persistence": {
      "enabled": true,
      "interval": "1m0s",
      "last_persist": "2024-01-01T10:30:00Z"
    },
    "memory": {
      "endpoints_tracked": 45,
      "max_endpoints": 100,
      "estimated_mb": 0.008
    },
    "endpoints": {
      "GET /api/v1/users": {
        "requests": 450,
        "errors": 12,
        "avg_time_ms": 125.5,
        "error_rate": 2.67
      }
    }
  }
}
```

### 🔥 Firebase Authentication

```go
// Setup Firebase Auth - extrai custom claims automaticamente
app.SetupFirebaseAuth(zendia.FirebaseAuthConfig{
    FirebaseClient: firebaseAuth,
    PublicRoutes:   []string{"/public", "/docs", "/auth"},
})

// 🎯 Use as CONSTANTES do framework para custom claims
import zendia "github.com/azzidev/zendiaframework"

claims := map[string]interface{}{
    zendia.ClaimTenantID: userFromDB.TenantID,  // ✅ Constante
    zendia.ClaimUserUUID: userFromDB.ID,        // ✅ Constante  
    zendia.ClaimUserName: userFromDB.Name,      // ✅ Constante
    zendia.ClaimRole:     userFromDB.Role,      // ✅ Do seu banco
}
// ❌ NÃO use strings: "tenant_id", "user_uuid", etc.

// Login PÚBLICO: email/senha → Firebase token + custom claims
app.POST("/auth/login", zendia.Handle(func(c *zendia.Context[any]) error {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    c.Context.ShouldBindJSON(&req)
    
    // 1. Autentica Firebase
    token, _ := authenticateFirebase(req.Email, req.Password)
    decodedToken, _ := firebaseAuth.VerifyIDToken(ctx, token)
    
    // 2. Busca no banco
    userFromDB := myUserRepo.FindByEmail(req.Email)
    
    // 3. Seta custom claims (PARA SEMPRE) - USE AS CONSTANTES!
    claims := map[string]interface{}{
        zendia.ClaimTenantID: userFromDB.TenantID,
        zendia.ClaimUserUUID: userFromDB.ID,
        zendia.ClaimUserName: userFromDB.Name,
        zendia.ClaimRole:     userFromDB.Role,
    }
    firebaseAuth.SetCustomUserClaims(ctx, decodedToken.UID, claims)
    
    c.Success("Login realizado", map[string]interface{}{
        "token": token,
    })
    return nil
}))

// Todas as rotas protegidas - framework extrai custom claims automaticamente
api := app.Group("/api/v1")

// Dados completos em qualquer handler
api.GET("/profile", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser()
        
    c.Success("Perfil do usuário", map[string]interface{}{
        "firebase_uid": user.FirebaseUID, // ← Do Firebase
        "email":        user.Email,       // ← Do Firebase
        "user_id":      user.ID,          // ← Custom claim
        "tenant":       user.TenantID,    // ← Custom claim
        "role":         user.Role,        // ← Custom claim (use conforme necessário)
    })
    return nil
})
```

### 🔄 **QueryOptions Avançado**

```go
// Ordenação complexa com múltiplos critérios
queryOpts := &zendia.QueryOptions{
    Sort: map[string]interface{}{
        "status": 1,           // Status crescente (active, inactive, etc.)
        "priority": -1,        // Prioridade decrescente (high, medium, low)
        "updated.set_at": -1,  // Mais recentemente atualizado
        "name": 1,             // Nome alfabético como último critério
    },
}

// Projeção - retornar apenas campos específicos
opts := &zendia.QueryOptions{
    Sort:       map[string]interface{}{"created.set_at": -1},
    Limit:      50,
    Projection: map[string]interface{}{"name": 1, "email": 1, "status": 1},
}

// Aplicação em diferentes cenários
// 1. Dashboard - tarefas por prioridade e data
tasks, err := taskRepo.GetAllSkipTake(ctx, 
    map[string]interface{}{"status": "active"}, 
    0, 50, 
    &zendia.QueryOptions{
        Sort: map[string]interface{}{
            "priority": -1,
            "due_date": 1,
        },
    },
)

// 2. Relatórios - ordenação por período
reports, err := reportRepo.GetAll(ctx, filters, &zendia.QueryOptions{
    Sort: map[string]interface{}{
        "created.set_at": -1, // Mais recente primeiro
    },
})

// 3. Auditoria - histórico cronológico
auditLogs, err := auditRepo.GetAllSkipTake(ctx, 
    map[string]interface{}{"entity_id": entityID}, 
    0, 100, 
    &zendia.QueryOptions{
        Sort: map[string]interface{}{
            "trigger.at": -1, // Mais recente primeiro
        },
    },
)
```

### 🔧 Configuração Avançada de Monitoring

```go
// Configuração customizada de métricas
config := zendia.MetricsConfig{
    MaxEndpoints:      200,                // Máx endpoints rastreados
    CleanupInterval:   10 * time.Minute,   // Limpeza a cada 10min
    MaxMemoryMB:      20,                  // Máx 20MB de memória
    PersistInterval:   30 * time.Second,   // Salva a cada 30s
    EnablePersistence: true,               // Habilita persistência
}

metrics := zendia.NewMetricsWithConfig(config)

// Configura persistidor customizado
if mongoAvailable {
    persister := zendia.NewMongoMetricsPersister(metricsCollection)
    metrics.SetPersister(persister)
}

app.Use(zendia.Monitoring(metrics))

// Consultar histórico programaticamente
history, err := metrics.GetMetricsHistory("tenant-123", 
    time.Now().Add(-24*time.Hour), time.Now())
```

### 🔐 Segurança Adicional

```go
// Rate limiting
app.Use(zendia.RateLimiter(100, time.Minute))

// CORS configurável
app.Use(zendia.CORS())
```

### 📚 Documentação Automática (Sem Navbar!)

```go
app.SetupSwagger(zendia.SwaggerInfo{
    Title:       "My API",
    Description: "API com ZendiaFramework",
    Version:     "1.0",
})

// @Summary Create user
// @Description Creates a new user with automatic audit
// @Tags users
// @Accept json
// @Produce json
// @Param user body User true "User data"
// @Success 201 {object} User
// @Router /users [post]
func createUser(c *zendia.Context[User]) error {
    var user User
    if err := c.BindJSON(&user); err != nil {
        return err // Validação em português automática!
    }
    // Auditoria e tenant automáticos
    created, err := userRepo.Create(c.Request.Context(), &user)
    c.Created(created)
    return nil
}
```

---

## 🎯 Casos de Uso

### ✅ Perfeito Para:

- 🏢 **APIs Multi-tenant** - SaaS, B2B, plataformas
- 📊 **Sistemas com Auditoria** - Compliance, rastreabilidade
- 🔄 **Microserviços** - Observabilidade e health checks
- 🚀 **MVPs Rápidos** - Setup mínimo, máxima produtividade
- 🏗️ **APIs Corporativas** - Padrões, segurança, monitoramento

### 🎩 Casos Reais com Firebase Auth:

```go
// E-commerce multi-tenant
api.POST("/orders", zendia.Handle(func(c *zendia.Context[Order]) error {
    user := c.GetAuthUser()   // Firebase data
    tenantID := c.GetTenantID() // ← Da sessão (login)
    
    if tenantID == "" {
        return zendia.NewBadRequestError("Faça login primeiro")
    }
    
    // Auditoria automática com tenant da sessão
}))

// Sistema bancário - implemente suas próprias permissões
api.PUT("/accounts/:id", zendia.Handle(func(c *zendia.Context[Account]) error {
    user := c.GetAuthUser()
    tenantID := c.GetTenantID() // ← Da sessão
    
    // Exemplo: if !userHasPermission(user.ID, "account:update") { return Forbidden }
    
    log.Printf("User %s (tenant: %s) updating account", user.Email, tenantID)
    // Todas as alterações auditadas automaticamente
}))

// Plataforma SaaS - dados por tenant
api.GET("/analytics", zendia.Handle(func(c *zendia.Context[any]) error {
    tenantID := c.GetTenantID() // ← Da sessão, não do Firebase
    
    // Dados filtrados automaticamente por tenant
    analytics := analyticsRepo.GetByTenant(tenantID)
    c.Success("Analytics", analytics)
    return nil
}))

// Dashboard - implemente suas próprias permissões
api.GET("/dashboard", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser()
    // Exemplo: if !userHasPermission(user.ID, "dashboard:view") { return Forbidden }
    
    c.Success("Dashboard data", map[string]string{"message": "Dashboard"})
    return nil
})
```

---

## 📈 Performance & Observabilidade

### Métricas Automáticas

- ⏱️ **Response Time** por endpoint
- 📊 **Request Count** e **Error Rate**
- 🔄 **Active Requests** em tempo real
- 📈 **Throughput** e estatísticas detalhadas

### Health Checks Reais (Sem Mocks!)

```bash
# Global - Memória real + Disco real
GET /health

# Por componente - Testes reais de conectividade  
GET /api/v1/health          # HTTP requests + MongoDB ping
GET /api/v1/users/health    # Repository operations reais
```

**Exemplo de resposta com dados reais:**

```json
{
  "status": "UP",
  "checks": {
    "memory": {
      "status": "UP",
      "details": {
        "alloc_mb": 45,
        "heap_mb": 32, 
        "goroutines": 8,
        "gc_cycles": 12
      }
    },
    "mongodb": {
      "status": "UP",
      "details": {
        "response_time_ms": 23
      }
    },
    "firebase_auth": {
      "status": "UP",
      "details": {
        "active_users": 1247,
        "auth_requests_today": 8934
      }
    }
  }
}
```

### 🔥 Firebase Auth Features

- ✅ **Token Validation** automática
- ✅ **Email/Password** provider support
- ✅ **Custom Claims** extraídos automaticamente
- ✅ **Multi-tenant** com custom claims

- ✅ **Public Routes** configuráveis
- ✅ **Context Integration** com `c.GetAuthUser()`
- ✅ **Auditoria Automática** com tenant/user_id
- ✅ **Error Handling** padronizado

---

## 🧪 Testes

```bash
# Executar todos os testes
go test ./...

# Com coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Testes específicos
go test -v ./repository_test.go
```

### Exemplo de Teste

```go
func TestUserCreation(t *testing.T) {
    app := zendia.New()
  
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"João"}`))
    req.Header.Set("X-Tenant-ID", "test-tenant")
    req.Header.Set("X-User-ID", "test-user")
  
    app.ServeHTTP(w, req)
  
    assert.Equal(t, 201, w.Code)
}
```

---

## 🚀 Exemplo Completo

Veja [`examples/complete_example.go`](examples/complete_example.go) para um exemplo completo com:

- ✅ CRUD completo com auditoria
- ✅ MongoDB + auditoria + histórico
- ✅ Repository unificado com options
- ✅ Autenticação e autorização
- ✅ Health checks granulares
- ✅ Métricas e monitoring
- ✅ Documentação Swagger
- ✅ Multi-tenant automático

```bash
# Executar exemplo
cd examples
go run example.go

# 1. Login (seta tenant na sessão)
curl -X POST -H "Authorization: Bearer <firebase-token>" http://localhost:8080/login

# 2. Testar dados do usuário (com tenant da sessão)
curl -H "Authorization: Bearer <firebase-token>" http://localhost:8080/api/v1/me

# 3. Criar usuário (usa tenant automaticamente)
curl -X POST -H "Authorization: Bearer <firebase-token>" \
     -H "Content-Type: application/json" \
     -d '{"name":"João","email":"joao@test.com","age":30}' \
     http://localhost:8080/api/v1/users

# 4. Métricas em tempo real (sem token)
curl http://localhost:8080/public/metrics

# 5. Histórico de métricas (se MongoDB disponível)
curl http://localhost:8080/public/metrics/history

# 6. Estatísticas agregadas por hora
curl "http://localhost:8080/public/metrics/stats?interval=hour"

# 7. Health checks granulares
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/users/health
```

### 🔧 Setup Firebase

1. **Crie um projeto** no [Firebase Console](https://console.firebase.google.com)
2. **Ative Authentication** → Email/Password
3. **Baixe o Service Account Key**:
   - Project Settings → Service Accounts → Generate New Private Key
4. **Configure as credenciais**:
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS="path/to/serviceAccountKey.json"
   ```
5. **Fluxo Correto**:
   ```bash
   # 1. Firebase só autentica (email/password)
   # 2. POST /login com token → Dev seta tenant do banco
   # 3. Próximas requests → Tenant automático da sessão
   ```
6. **Token Firebase** (com custom claims):
   ```json
   {
     "uid": "firebase-uid-123",
     "email": "user@example.com",
     "tenant_id": "company-123",
     "user_uuid": "user-456",
     "user_name": "John Doe",
     "role": "admin"
   }
   ```

---

## 📋 Padrões Obrigatórios

### 🗄️ **Estrutura de Collections MongoDB**

O framework **recomenda** padrões específicos para garantir consistência e segurança:

#### **Collections Padrão**

```go
// Collections padrão (você pode usar qualquer nome)
const (
    DefaultUsersCollection   = "users"    // Collection principal (configurável)
    DefaultHistoryCollection = "history"  // Collection de histórico (configurável)
)
```

#### **Estrutura de Entidade Obrigatória**

```go
type User struct {
    // ✅ OBRIGATÓRIO - ID como UUID
    ID        uuid.UUID `bson:"_id" json:"id"`
    
    // ✅ OBRIGATÓRIO - Tenant para multi-tenancy
    TenantID  uuid.UUID `bson:"tenant_id" json:"tenant_id"`
    
    // ✅ OBRIGATÓRIO - Flag de soft delete
    Active    bool      `bson:"active" json:"active"`
    
    // ✅ OBRIGATÓRIO - Auditoria com AuditInfo
    Created   zendia.AuditInfo `bson:"created" json:"created"`
    Updated   zendia.AuditInfo `bson:"updated" json:"updated"`
    Deleted   zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"`
    
    // Seus campos customizados
    Name      string    `json:"name" validate:"required,min=2,max=50"`
    Email     string    `json:"email" validate:"required,email"`
}

// ✅ OBRIGATÓRIO - Implementar interfaces
func (u *User) GetID() uuid.UUID         { return u.ID }
func (u *User) SetID(id uuid.UUID)       { u.ID = id }
func (u *User) SetTenantID(s string)     { u.TenantID = uuid.MustParse(s) }

// Para estrutura AuditInfo
func (u *User) SetCreated(info zendia.AuditInfo) { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo) { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo) { u.Deleted = info }
func (u *User) SetActive(active bool)            { /* implementar conforme necessário */ }
```

#### **Estrutura AuditInfo**

```go
type AuditInfo struct {
    SetAt  time.Time `bson:"set_at" json:"set_at"`     // Quando foi alterado
    ByName string    `bson:"by_name" json:"by_name"`   // Nome do usuário
    ByID   uuid.UUID `bson:"by_id" json:"by_id"`       // ID do usuário
}
```

#### **Collection de Histórico (Automática)**

```go
type HistoryEntry struct {
    ID          uuid.UUID              `bson:"_id" json:"id"`
    EntityID    uuid.UUID              `bson:"entity_id" json:"entityId"`
    EntityType  string                 `bson:"entity_type" json:"entityType"`
    TenantID    uuid.UUID              `bson:"tenant_id" json:"tenantId"`
    TriggerName string                 `bson:"trigger_name" json:"triggerName"`
    TriggerAt   time.Time              `bson:"trigger_at" json:"triggerAt"`
    TriggerBy   string                 `bson:"trigger_by" json:"triggerBy"`
    Changes     map[string]FieldChange `bson:"changes" json:"changes"`
}

type FieldChange struct {
    Before interface{} `bson:"before" json:"before"`
    After  interface{} `bson:"after" json:"after"`
}
```

### 🔍 **Input Sanitization Configurável**

O `InputSanitizer` protege contra NoSQL injection em input HTTP. **Filtros internos do dev não passam por sanitização.**

```go
// Cria sanitizador com campos customizados além dos padrão
sanitizer := zendia.NewInputSanitizer("project_id", "sprint_id", "category")

// Sanitiza input do usuário
safeInput, err := sanitizer.Sanitize(userInput)
```

Campos padrão permitidos: `_id`, `tenant_id`, `name`, `email`, `status`, `active`.

```go
// Whitelist é configurável por projeto
var allowedFilterKeys = map[string]bool{
    "_id":              true,
    "tenant_id":        true,
    "name":             true,
    "email":            true,
    "status":           true,
    "active":           true,
    // AuditInfo fields
    "created.set_at":    true,
    "created.by_name":   true,
    "created.by_id":     true,
    "created.active":    true,
    "updated.set_at":    true,
    "updated.by_name":   true,
    "updated.by_id":     true,
    "deleted.set_at":    true,
    "deleted.by_name":   true,
    "deleted.by_id":     true,
    "deleted.active":    true,
}
```

### ⚠️ **Regras Importantes**

1. **UUID Obrigatório**: Todos os IDs devem ser UUID v4
2. **TenantID Obrigatório**: Multi-tenancy é forçado
3. **Auditoria Opcional**: Use `WithAudit()` para habilitar
4. **Histórico Opcional**: Use `WithHistory()` para habilitar
4. **Collections Configuráveis**: Você escolhe os nomes (users/history são sugestões)
5. **Filtros Limitados**: Só campos na whitelist
6. **Paginação Limitada**: Máximo 1000 itens por página
7. **Validação Obrigatória**: Use tags `validate`

### 🚫 **O que NÃO Funciona**

```go
// ❌ ID como string ou int
type User struct {
    ID string `json:"id"` // ERRO!
}

// ❌ Sem TenantID
type User struct {
    Name string `json:"name"` // ERRO! Falta TenantID
}

// ❌ Filtros não permitidos
filters := map[string]interface{}{
    "$where": "function() { return true }", // ERRO! Bloqueado
    "password": "123",                      // ERRO! Não está na whitelist
}

// ❌ Paginação excessiva
take := 5000 // ERRO! Máximo é 1000
```

### ✅ **Setup Correto**

```go
// 1. Conecta MongoDB
client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

// 2. SEU banco e collections (você escolhe os nomes!)
db := client.Database("meu_projeto")  // ← SEU nome do banco
usersCollection := db.Collection("usuarios")     // ← SEU nome da collection
historyCollection := db.Collection("historico")  // ← SEU nome do histórico

// 3. Repository com histórico automático
userRepo := zendia.NewRepository[*User](
    usersCollection,
    zendia.WithAudit(),
    zendia.WithHistory(historyCollection, "User"),
)


```



---

## 🔒 Segurança

### 🎆 **ATUALIZAÇÃO v1.2.4 - Correção Crítica**

#### ✅ **Sanitização Corrigida**

**ANTES (v1.0.0 - v1.2.3):** ❌ Sanitização bloqueava código legítimo
```go
// ERRO: Bloqueava $or, $and do próprio dev!
filters := map[string]interface{}{
    "$or": []map[string]interface{}{ // ❌ REJEITADO!
        {"status": "active"},
        {"priority": "high"},
    },
}
```

**AGORA (v1.2.4+):** ✅ Sanitização APENAS no input do usuário
```go
// ✅ Código interno: SEM RESTRIÇÕES
filters := map[string]interface{}{
    "$or": []map[string]interface{}{ // ✅ FUNCIONA!
        {"status": "active"},
        {"priority": "high"},
    },
}

// ✅ Input HTTP: SANITIZADO automaticamente
func handler(c *zendia.Context[MyStruct]) error {
    var data MyStruct
    c.BindJSON(&data) // ← Sanitiza automaticamente!
    return nil
}
```

#### 🛡️ **Proteção Atual**

1. **NoSQL Injection Prevention**
   - ✅ **Input HTTP** (JSON/Query/URI) → Sanitizado automaticamente
   - ✅ **Código interno** → Livre para usar $or, $and, $regex, etc.
   - ✅ **Trust Boundary** → Separação correta entre input externo e código interno

2. **XSS Prevention** 
   - Sanitização de valores de headers HTTP
   - Escape automático de caracteres perigosos
   - Limitação de tamanho para prevenir DoS

3. **Log Injection Prevention**
   - Sanitização de valores antes do logging
   - Remoção de caracteres de controle
   - Logs de auditoria não manipuláveis

4. **Context Security**
   - Uso correto do request context
   - Propagação adequada de cancelamento
   - Prevenção de vazamento de goroutines

### Configuração Segura

```bash
# Variáveis de ambiente obrigatórias
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/serviceAccountKey.json"
export FIREBASE_PROJECT_ID="your-project-id"

# Configurações opcionais de segurança
export ZENDIA_MAX_FILTERS="20"
export ZENDIA_MAX_PAGINATION="1000"
export ZENDIA_LOG_LEVEL="INFO"
```

### Boas Práticas

```go
// ✅ Input do usuário: Sempre usar BindJSON/BindQuery/BindURI
func createUser(c *zendia.Context[CreateUserRequest]) error {
    var req CreateUserRequest
    if err := c.BindJSON(&req); err != nil {
        return err // ← Já sanitizado automaticamente!
    }
    // req agora é seguro para usar
}

// ✅ Filtros internos: Use livremente operadores MongoDB
filters := map[string]interface{}{
    "$or": []map[string]interface{}{ // ✅ FUNCIONA!
        {"status": "active"},
        {"name": bson.M{"$regex": "^John"}}, // ✅ FUNCIONA!
    },
    "$and": []map[string]interface{}{ // ✅ FUNCIONA!
        {"age": bson.M{"$gte": 18}},
        {"verified": true},
    },
}

// ✅ Paginação com limites
if skip < 0 || take < 0 || take > 1000 {
    return NewBadRequestError("Invalid pagination")
}

// ✅ Contexto de tenant sempre validado
user := c.GetAuthUser()
if user.TenantID == "" {
    c.Unauthorized("Invalid tenant context")
    return nil
}

// ✅ Use as constantes do framework
claims := map[string]interface{}{
    zendia.ClaimTenantID: userFromDB.TenantID,  // ✅
    zendia.ClaimUserUUID: userFromDB.ID,        // ✅
    // ❌ NÃO: "tenant_id": userFromDB.TenantID
}
```

### Checklist de Segurança

- [ ] Variáveis de ambiente configuradas
- [ ] Firebase credentials seguras
- [ ] Validação em todos os endpoints
- [ ] Rate limiting configurado
- [ ] HTTPS enforçado em produção
- [ ] Logs de auditoria habilitados
- [ ] Headers de segurança configurados
- [ ] Tenant isolation testado

---

## 📄 Licença

Distribuído sob a licença MIT. Veja `LICENSE` para mais informações.

---

## 🙏 Agradecimentos

- [Gin](https://github.com/gin-gonic/gin) - Framework HTTP base
- [Go Playground Validator](https://github.com/go-playground/validator) - Validação de dados
- [MongoDB Driver](https://github.com/mongodb/mongo-go-driver) - Driver MongoDB oficial
- [Google UUID](https://github.com/google/uuid) - Geração de UUIDs

---

<div align="center">
  <p>Feito com ❤️ para a comunidade Go brasileira</p>
  <p><strong>ZendiaFramework</strong> - Simplicidade que escala 🚀</p>
</div>
