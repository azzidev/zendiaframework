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
- 🚨 **Error Handling** - Tratamento padronizado e consistente de erros
- 📝 **Auditoria** - Tracking automático de criação/modificação

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
    
    // Setup Firebase Auth
    app.SetupAuth(zendia.AuthConfig{
        FirebaseClient: firebaseAuth,
        PublicRoutes:   []string{"/public"},
    })
  
    // Rota protegida automaticamente
    api := app.Group("/api/v1")
    api.GET("/hello", zendia.Handle(func(c *zendia.Context[any]) error {
        user := c.GetAuthUser() // Dados do usuário autenticado
        c.Success(map[string]interface{}{
            "message": "Hello from ZendiaFramework! 🎉",
            "user":    user.Name,
            "email":   user.Email,
            "tenant":  c.GetTenantID(),
        })
        return nil
    }))
    
    // Rota pública
    app.GET("/public/status", zendia.Handle(func(c *zendia.Context[any]) error {
        c.Success(map[string]string{"status": "ok"})
        return nil
    }))
  
    app.Run(":8080")
}
```

### Teste com Firebase Token

```bash
# Rota protegida (precisa de token)
curl -H "Authorization: Bearer <firebase-token>" http://localhost:8080/api/v1/hello

# Rota pública (sem token)
curl http://localhost:8080/public/status
```

---

## 🏗️ Arquitetura

### Multi-Tenant por Padrão

```go
// Contexto de tenant automático em TODAS as requisições
func createUser(c *zendia.Context[User]) error {
    // TenantID, UserID e ActionAt já disponíveis!
    tenantID := c.GetTenantID()  // Automático
    userID := c.GetUserID()      // Automático
  
    var user User
    c.BindJSON(&user) // Validação automática
  
    // Auditoria automática (CreatedBy, CreatedAt, TenantID)
    created, err := userRepo.Create(c.Request.Context(), &user)
  
    c.Created(created)
    return nil
}
```

### Repository com Auditoria

```go
import "github.com/google/uuid"

type User struct {
    ID        uuid.UUID `bson:"_id" json:"id"`
    Name      string    `json:"name" validate:"required,min=2"`
    Email     string    `json:"email" validate:"required,email"`
    TenantID  uuid.UUID `bson:"tenant_id" json:"tenant_id"` // Preenchido automaticamente
    CreatedAt time.Time `bson:"created_at" json:"created_at"` // Preenchido automaticamente
    CreatedBy string    `bson:"created_by" json:"created_by"` // Preenchido automaticamente
    UpdatedAt time.Time `bson:"updated_at" json:"updated_at"` // Atualizado automaticamente
    UpdatedBy string    `bson:"updated_by" json:"updated_by"` // Atualizado automaticamente
}

// Implementa interface para auditoria automática
func (u *User) GetID() uuid.UUID        { return u.ID }
func (u *User) SetID(id uuid.UUID)      { u.ID = id }
func (u *User) SetCreatedAt(t time.Time) { u.CreatedAt = t }
func (u *User) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }
func (u *User) SetCreatedBy(s string)    { u.CreatedBy = s }
func (u *User) SetUpdatedBy(s string)    { u.UpdatedBy = s }
func (u *User) SetTenantID(s string)     { u.TenantID = uuid.MustParse(s) }

// MongoDB com UUID nativo - auditoria automática!
baseRepo := zendia.NewMongoAuditRepository[*User](collection)
// ou
baseRepo := zendia.NewAuditRepository[*User, uuid.UUID](memoryRepo)
```

---

## 🛠️ Funcionalidades Avançadas

### 📊 Monitoramento Completo

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

// Health por grupo
users := app.Group("/users")
usersHealth := zendia.NewHealthManager()
usersHealth.AddCheck(zendia.NewMemoryHealthCheck(1024))
users.AddHealthEndpoint(usersHealth) // GET /users/health
```

### 🔥 Firebase Authentication

```go
// Setup Firebase Auth uma vez
opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
firebaseApp, err := firebase.NewApp(context.Background(), nil, opt)
if err != nil {
    log.Fatal("Firebase init failed:", err)
}
firebaseAuth, err := firebaseApp.Auth(context.Background())
if err != nil {
    log.Fatal("Firebase Auth init failed:", err)
}

app.SetupAuth(zendia.AuthConfig{
    FirebaseClient: firebaseAuth,
    PublicRoutes:   []string{"/public", "/docs"},
})

// Todas as rotas são protegidas automaticamente
api := app.Group("/api/v1") // Já protegido!

// Roles específicas
adminRoutes := api.Group("/admin").RequireRole("admin")
managerRoutes := api.Group("/management").RequireRole("admin", "manager")

// Email verificado obrigatório
verifiedRoutes := api.Group("/verified", zendia.RequireEmailVerified())

// Dados do usuário em qualquer handler
api.GET("/profile", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser()
    if c.HasRole("admin") {
        // Lógica para admin
    }
    c.Success(user)
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

### 📚 Documentação Automática

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
    // Implementação
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
    user := c.GetAuthUser() // Usuário autenticado
    // TenantID automático, UserID do Firebase
    // Auditoria automática para compliance
}))

// Sistema bancário - só gerentes
managerRoutes.PUT("/accounts/:id", zendia.Handle(func(c *zendia.Context[Account]) error {
    user := c.GetAuthUser()
    log.Printf("Manager %s updating account", user.Email)
    // Todas as alterações auditadas automaticamente
}))

// Plataforma SaaS - dados por tenant
api.GET("/analytics", zendia.Handle(func(c *zendia.Context[any]) error {
    user := c.GetAuthUser()
    tenantID := user.TenantID // Do token Firebase
    // Dados filtrados automaticamente por tenant
    // Métricas de uso por cliente autenticado
}))

// Admin dashboard - só admins
adminRoutes.GET("/dashboard", zendia.Handle(func(c *zendia.Context[any]) error {
    // Acesso garantido apenas para role 'admin'
    c.Success(map[string]string{"message": "Admin data"})
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

# Testar rota pública
curl http://localhost:8080/public/metrics

# Testar rota protegida (precisa de Firebase token)
curl -H "Authorization: Bearer <firebase-token>" http://localhost:8080/api/v1/me

# Testar rota admin (precisa de role 'admin' no token)
curl -H "Authorization: Bearer <admin-firebase-token>" http://localhost:8080/api/v1/admin/stats
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
5. **Token Structure** no Firebase deve ter:
   ```json
   {
     "email": "user@example.com",
     "name": "User Name",
     "role": "admin",
     "tenant_id": "company-123"
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
