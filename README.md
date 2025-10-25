<div align="center">
  <h1>🚀 ZendiaFramework</h1>
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
- 🗄️ **Repository Pattern** - Suporte a MongoDB e in-memory com auditoria automática
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
- 📈 **Observabilidade** - Tracing distribuído e métricas detalhadas
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
    
    // Setup Firebase Auth (só autentica)
    app.SetupFirebaseAuth(zendia.FirebaseAuthConfig{
        FirebaseClient: firebaseAuth,
        PublicRoutes:   []string{"/public", "/login"},
    })
    
    // Login: Dev seta tenant manualmente
    app.POST("/login", zendia.Handle(func(c *zendia.Context[any]) error {
        user := c.GetAuthUser() // Dados do Firebase
        
        // Dev busca no SEU banco
        // userFromDB := myRepo.FindByEmail(user.Email)
        
        // Seta tenant na sessão
        c.SetTenant("company-123")  // ← Do seu banco
        c.SetUserID("user-456")     // ← ID do seu sistema
        c.SetRole("admin")          // ← Role do seu sistema
        
        c.Success("Login realizado", map[string]interface{}{
            "user":     user,
            "tenant":   c.GetTenantID(),
        })
        return nil
    }))
  
    // Rota protegida - tenant automático da sessão
    api := app.Group("/api/v1")
    api.GET("/hello", zendia.Handle(func(c *zendia.Context[any]) error {
        user := c.GetAuthUser()
        tenantID := c.GetTenantID() // ← Da sessão, não do Firebase
        
        c.Success("Hello from ZendiaFramework! 🎉", map[string]interface{}{
            "user":    user.Name,
            "email":   user.Email,
            "tenant":  tenantID,
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

### Teste com Firebase Token

```bash
# 1. Login (seta tenant na sessão)
curl -X POST -H "Authorization: Bearer <firebase-token>" http://localhost:8080/login

# 2. Rota protegida (usa tenant da sessão)
curl -H "Authorization: Bearer <firebase-token>" http://localhost:8080/api/v1/hello

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

### Repository com Auditoria e Histórico

```go
import "github.com/google/uuid"

type User struct {
    ID        uuid.UUID `bson:"_id" json:"id"`
    Name      string    `json:"name" validate:"required,min=2"`
    Email     string    `json:"email" validate:"required,email"`
    TenantID  uuid.UUID `bson:"tenant_id" json:"tenant_id"` // Preenchido automaticamente
    Created   zendia.AuditInfo `bson:"created" json:"created"`   // Nova interface de auditoria
    Updated   zendia.AuditInfo `bson:"updated" json:"updated"`   // Nova interface de auditoria
    DeletedAt *time.Time `bson:"deleted_at" json:"deletedAt,omitempty"`
    DeletedBy string     `bson:"deleted_by" json:"deletedBy,omitempty"`
}

// Implementa interface para auditoria automática
func (u *User) GetID() uuid.UUID              { return u.ID }
func (u *User) SetID(id uuid.UUID)            { u.ID = id }
func (u *User) SetCreated(info zendia.AuditInfo) { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo) { u.Updated = info }
func (u *User) SetTenantID(s string)          { u.TenantID = uuid.MustParse(s) }

// Repository com auditoria e histórico automático!
repo := zendia.NewHistoryAuditRepository[*User](collection, historyCollection, "User")
// ou apenas auditoria
repo := zendia.NewMongoAuditRepository[*User](collection)
```

---

## 🛠️ Funcionalidades Avançadas

### 📊 Monitoramento e Histórico Completo

```go
app := zendia.New()

// Métricas automáticas
metrics := app.AddMonitoring()

// Tracing distribuído
tracer := zendia.NewSimpleTracer()
app.Use(zendia.Tracing(tracer))

// Health checks granulares
globalHealth := zendia.NewHealthManager()
globalHealth.AddCheck(zendia.NewDatabaseHealthCheck("main_db", dbPing))
app.AddHealthEndpoint(globalHealth) // GET /health

// Repository com histórico automático
projectRepo := zendia.NewHistoryAuditRepository[*Project](
    db.Collection("projects"),
    db.Collection("history"),
    "Project",
)

// Histórico automático em updates
project, err := projectRepo.Update(ctx, id, updatedProject)
// Registra automaticamente apenas os campos que mudaram!

// Consultar histórico
history, err := projectRepo.GetHistory(ctx, projectID)
// Retorna: [{"Name": {"before": "Old", "after": "New"}}]
```

### 🔥 Firebase Authentication

```go
// Setup Firebase Auth (só autentica)
app.SetupFirebaseAuth(zendia.FirebaseAuthConfig{
    FirebaseClient: firebaseAuth,
    PublicRoutes:   []string{"/public", "/docs", "/login"},
})

// Login obrigatório para setar tenant
app.POST("/login", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser() // Dados do Firebase
    
    // Dev busca dados no SEU banco
    userFromDB := myUserRepo.FindByEmail(user.Email)
    
    // Seta dados na sessão
    c.SetTenant(userFromDB.TenantID)
    c.SetUserID(userFromDB.ID)
    c.SetRole(userFromDB.Role)
    
    c.Success("Login realizado", user)
    return nil
}))

// Todas as rotas protegidas usam dados da sessão
api := app.Group("/api/v1")

// Roles específicas (setadas no login)
adminRoutes := api.Group("/admin", zendia.RequireRole("admin"))
managerRoutes := api.Group("/management", zendia.RequireRole("admin", "manager"))

// Email verificado (do Firebase)
verifiedRoutes := api.Group("/verified", zendia.RequireEmailVerified())

// Dados completos em qualquer handler
api.GET("/profile", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser()
    tenantID := c.GetTenantID() // ← Da sessão
    
    if c.HasRole("admin") {
        // Lógica para admin
    }
    
    c.Success("Perfil do usuário", map[string]interface{}{
        "user":     user,
        "tenant":   tenantID,
    })
    return nil
}))
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

// Sistema bancário - só gerentes
managerRoutes.PUT("/accounts/:id", zendia.Handle(func(c *zendia.Context[Account]) error {
    user := c.GetAuthUser()
    tenantID := c.GetTenantID() // ← Da sessão
    
    log.Printf("Manager %s (tenant: %s) updating account", user.Email, tenantID)
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

// Admin dashboard - só admins
adminRoutes.GET("/dashboard", zendia.Handle(func(c *zendia.Context[any]) error {
    // Role setada no login, não no Firebase
    c.Success("Admin data", map[string]string{"message": "Admin dashboard"})
    return nil
}))
```

---

## 📈 Performance & Observabilidade

### Métricas Automáticas

- ⏱️ **Response Time** por endpoint
- 📊 **Request Count** e **Error Rate**
- 🔄 **Active Requests** em tempo real
- 📈 **Throughput** e estatísticas detalhadas

### Tracing Distribuído

- 🔍 **Trace ID** automático em todas as requisições
- 📝 **Spans** com contexto completo
- 🔗 **Propagação** entre serviços
- 📊 **Visualização** de performance

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
- ✅ **User Data** extraído do token (email, name, picture)
- ✅ **Role-based Access** com `RequireRole()`
- ✅ **Email Verification** com `RequireEmailVerified()`
- ✅ **Multi-tenant** com tenant_id no token
- ✅ **Public Routes** configuráveis
- ✅ **Context Integration** com `c.GetAuthUser()`
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
- ✅ MongoDB + fallback in-memory
- ✅ Autenticação e autorização
- ✅ Health checks granulares
- ✅ Métricas e tracing
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

# 4. Rota pública (sem token)
curl http://localhost:8080/public/metrics
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
6. **Token Firebase** (padrão, sem custom claims):
   ```json
   {
     "email": "user@example.com",
     "name": "User Name",
     "email_verified": true
   }
   ```

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
