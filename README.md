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
    Deleted   zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"` // Consistente com AuditInfo
}

// Implementa interface para auditoria automática
func (u *User) GetID() uuid.UUID              { return u.ID }
func (u *User) SetID(id uuid.UUID)            { u.ID = id }
func (u *User) SetCreated(info zendia.AuditInfo) { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo) { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo) { u.Deleted = info }
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
    
    // ✅ OBRIGATÓRIO - Auditoria (escolha uma das opções)
    // Opção 1: Nova estrutura AuditInfo
    Created   zendia.AuditInfo `bson:"created" json:"created"`
    Updated   zendia.AuditInfo `bson:"updated" json:"updated"`
    Deleted   zendia.AuditInfo `bson:"deleted" json:"deleted,omitempty"`
    
    // Opção 2: Estrutura legacy (compatibilidade)
    CreatedAt time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
    CreatedBy string    `bson:"created_by" json:"created_by"`
    UpdatedBy string    `bson:"updated_by" json:"updated_by"`
    DeletedAt *time.Time `bson:"deleted_at" json:"deleted_at,omitempty"`
    DeletedBy string     `bson:"deleted_by" json:"deleted_by,omitempty"`
    
    // Seus campos customizados
    Name      string    `json:"name" validate:"required,min=2,max=50"`
    Email     string    `json:"email" validate:"required,email"`
}

// ✅ OBRIGATÓRIO - Implementar interfaces
func (u *User) GetID() uuid.UUID         { return u.ID }
func (u *User) SetID(id uuid.UUID)       { u.ID = id }
func (u *User) SetTenantID(s string)     { u.TenantID = uuid.MustParse(s) }

// Para nova estrutura AuditInfo
func (u *User) SetCreated(info zendia.AuditInfo) { u.Created = info }
func (u *User) SetUpdated(info zendia.AuditInfo) { u.Updated = info }
func (u *User) SetDeleted(info zendia.AuditInfo) { u.Deleted = info }

// Para estrutura legacy
func (u *User) SetCreatedAt(t time.Time) { u.CreatedAt = t }
func (u *User) SetUpdatedAt(t time.Time) { u.UpdatedAt = t }
func (u *User) SetCreatedBy(s string)    { u.CreatedBy = s }
func (u *User) SetUpdatedBy(s string)    { u.UpdatedBy = s }
```

#### **Estrutura AuditInfo (Recomendada)**

```go
type AuditInfo struct {
    SetAt  time.Time `bson:"set_at" json:"set_at"`     // Quando foi alterado
    ByName string    `bson:"by_name" json:"by_name"`   // Nome do usuário
    ByID   uuid.UUID `bson:"by_id" json:"by_id"`       // ID do usuário
    Active bool      `bson:"active" json:"active"`     // Se está ativo
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

### 🔍 **Campos de Filtro Permitidos**

O framework **só permite** filtros em campos seguros:

```go
// Whitelist de campos permitidos para filtros
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
    // Legacy fields (compatibilidade)
    "created_at":        true,
    "updated_at":        true,
    "deleted_at":        true,
    "created_by":        true,
    "updated_by":        true,
    "deleted_by":        true,
}
```

### ⚠️ **Regras Importantes**

1. **UUID Obrigatório**: Todos os IDs devem ser UUID v4
2. **TenantID Obrigatório**: Multi-tenancy é forçado
3. **Auditoria Obrigatória**: Escolha AuditInfo ou legacy
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
userRepo := zendia.NewHistoryAuditRepository[*User](
    usersCollection,
    historyCollection,
    "User", // Nome da entidade para histórico
)

// 4. Repository simples (só auditoria)
// userRepo := zendia.NewMongoAuditRepository[*User](usersCollection)
```



---

## 🔒 Segurança

### Correções Aplicadas

#### ✅ **Vulnerabilidades Críticas Corrigidas**

1. **NoSQL Injection Prevention**
   - Sanitização automática de filtros MongoDB
   - Validação de nomes de campos com whitelist
   - Proteção contra operadores MongoDB maliciosos

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
// ✅ Validação de entrada sempre
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email,max=255"`
}

// ✅ Filtros seguros com whitelist
allowedFilters := map[string]bool{
    "status": true,
    "name":   true,
    "email":  true,
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
