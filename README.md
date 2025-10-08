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
```

### Hello World
```go
package main

import "github.com/azzidev/zendiaframework"

func main() {
    app := zendia.New()
    
    app.GET("/hello", zendia.Handle(func(c *zendia.Context[any]) error {
        c.Success(map[string]string{
            "message": "Hello from ZendiaFramework! 🎉",
            "tenant":  c.GetTenantID(),
        })
        return nil
    }))
    
    app.Run(":8080")
}
```

### Teste rápido
```bash
curl -H "X-Tenant-ID: demo" -H "X-User-ID: user1" http://localhost:8080/hello
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
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name" validate:"required,min=2"`
    Email     string    `json:"email" validate:"required,email"`
    TenantID  string    `json:"tenant_id"`  // Preenchido automaticamente
    CreatedAt time.Time `json:"created_at"` // Preenchido automaticamente
    CreatedBy string    `json:"created_by"` // Preenchido automaticamente
    UpdatedAt time.Time `json:"updated_at"` // Atualizado automaticamente
    UpdatedBy string    `json:"updated_by"` // Atualizado automaticamente
}

// MongoDB ou In-Memory - mesma interface!
baseRepo := zendia.NewMongoAuditRepository[*User](collection)
// ou
baseRepo := zendia.NewAuditRepository[*User, string](memoryRepo)
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

### 🔒 Segurança Integrada
```go
// Rate limiting
app.Use(zendia.RateLimiter(100, time.Minute))

// CORS configurável
app.Use(zendia.CORS())

// Auth flexível
users := api.Group("/users", zendia.Auth(func(token string) bool {
    return validateJWT(token) // Sua lógica de validação
}))
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

### 🛡️ Casos Reais:
```go
// E-commerce multi-tenant
app.POST("/api/v1/orders", zendia.Handle(func(c *zendia.Context[Order]) error {
    // TenantID = loja, UserID = cliente
    // Auditoria automática para compliance
}))

// Sistema bancário
app.PUT("/api/v1/accounts/:id", zendia.Handle(func(c *zendia.Context[Account]) error {
    // Todas as alterações auditadas automaticamente
    // Health checks para cada componente crítico
}))

// Plataforma SaaS
app.GET("/api/v1/analytics", zendia.Handle(func(c *zendia.Context[any]) error {
    // Dados filtrados automaticamente por tenant
    // Métricas de uso por cliente
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
    }

  }
}
```

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
go run complete_example.go

# Testar
curl -H "X-Tenant-ID: demo" -H "X-User-ID: user1" http://localhost:8080/tenant-info
```

---

## 🤝 Contribuindo

1. Fork o projeto
2. Crie sua feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

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