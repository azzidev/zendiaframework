# ZendiaFramework

Um framework Go modular baseado no Gin para criação de APIs RESTful com foco em simplicidade e flexibilidade.

## Características

- **Roteamento Flexível**: Sistema de roteamento intuitivo com suporte a grupos
- **Middleware**: Suporte completo a middleware para funcionalidades transversais
- **Validação de Dados**: Validação automática com mensagens personalizadas
- **Gerenciamento de Erros**: Sistema padronizado de tratamento de erros
- **Documentação Automática**: Integração com Swagger para documentação da API
- **Tipos Genéricos**: Suporte completo a generics para flexibilidade
- **Modular**: Arquitetura modular permitindo customização

## Instalação

```bash
go get github.com/azzidev/zendiaframework
```

## Uso Básico

```go
package main

import "github.com/azzidev/zendiaframework"

func main() {
    app := zendia.New()
    
    // Middlewares
    app.Use(zendia.Logger())
    app.Use(zendia.CORS())
    
    // Rotas
    app.GET("/health", zendia.Handle(healthCheck))
    
    app.Run(":8080")
}

func healthCheck(c *zendia.Context[any]) error {
    c.Success(map[string]string{"status": "ok"})
    return nil
}
```

## Middlewares Disponíveis

- `Logger()`: Logging de requisições
- `CORS()`: Cross-Origin Resource Sharing
- `Compression()`: Compressão gzip
- `Auth(validator)`: Autenticação por token
- `RateLimiter(requests, window)`: Rate limiting

## Validação de Dados

```go
type User struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=120"`
}

func createUser(c *zendia.Context[User]) error {
    var user User
    if err := c.BindJSON(&user); err != nil {
        return err // Retorna erro de validação automaticamente
    }
    
    c.Created(user)
    return nil
}
```

## Grupos de Rotas

```go
api := app.Group("/api/v1")
users := api.Group("/users", zendia.Auth(tokenValidator))

users.GET("/", zendia.Handle(getUsers))
users.POST("/", zendia.Handle(createUser))
```

## Tratamento de Erros

```go
func getUser(c *zendia.Context[any]) error {
    id := c.Param("id")
    if id == "" {
        return zendia.NewBadRequestError("ID is required")
    }
    
    user := findUser(id)
    if user == nil {
        return zendia.NewNotFoundError("User not found")
    }
    
    c.Success(user)
    return nil
}
```

## Documentação Swagger

```go
app.SetupSwagger(zendia.SwaggerInfo{
    Title:       "My API",
    Description: "API documentation",
    Version:     "1.0",
    Host:        "localhost:8080",
    BasePath:    "/api/v1",
})

// @Summary Get user
// @Description Get user by ID
// @Tags users
// @Param id path int true "User ID"
// @Success 200 {object} User
// @Router /users/{id} [get]
func getUser(c *zendia.Context[any]) error {
    // implementação
}
```

## Testes

Execute os testes:

```bash
go test ./...
```

## Exemplo Completo

Veja o arquivo `examples/main.go` para um exemplo completo com CRUD de usuários.

## Licença

MIT License