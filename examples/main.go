package main

import (
	"time"

	"github.com/azzidev/zendiaframework"
)

// User representa um usuário
type User struct {
	ID    int    `json:"id" validate:"required"`
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=120"`
}

// CreateUserRequest representa a requisição de criação de usuário
type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=120"`
}

func main() {
	// Cria uma nova instância do framework
	app := zendia.New()

	// Adiciona middlewares globais
	app.Use(zendia.Logger())
	app.Use(zendia.CORS())
	app.Use(zendia.Compression())

	// Configura documentação Swagger
	app.SetupSwagger(zendia.SwaggerInfo{
		Title:       "Zendia Framework API",
		Description: "API de exemplo usando ZendiaFramework",
		Version:     "1.0",
		Host:        "localhost:8080",
		BasePath:    "/api/v1",
	})

	// Grupo de rotas da API
	api := app.Group("/api/v1")

	// Grupo de usuários com autenticação
	users := api.Group("/users", zendia.Auth(func(token string) bool {
		return token == "valid-token" // Validação simples para exemplo
	}))

	// Rotas de usuários
	users.GET("/", zendia.Handle(getUsers))
	users.GET("/:id", zendia.Handle(getUserByID))
	users.POST("/", zendia.Handle(createUser))
	users.PUT("/:id", zendia.Handle(updateUser))
	users.DELETE("/:id", zendia.Handle(deleteUser))

	// Rota pública
	app.GET("/health", zendia.Handle(healthCheck))

	// Inicia o servidor
	app.Run(":8080")
}

// @Summary Get all users
// @Description Get list of all users
// @Tags users
// @Produce json
// @Success 200 {array} User
// @Router /users [get]
func getUsers(c *zendia.Context[any]) error {
	users := []User{
		{ID: 1, Name: "João", Email: "joao@example.com", Age: 30},
		{ID: 2, Name: "Maria", Email: "maria@example.com", Age: 25},
	}
	c.Success(users)
	return nil
}

// @Summary Get user by ID
// @Description Get a user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} User
// @Failure 404 {object} zendia.APIError
// @Router /users/{id} [get]
func getUserByID(c *zendia.Context[any]) error {
	id := c.Param("id")
	if id == "1" {
		user := User{ID: 1, Name: "João", Email: "joao@example.com", Age: 30}
		c.Success(user)
		return nil
	}
	return zendia.NewNotFoundError("User not found")
}

// @Summary Create user
// @Description Create a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User data"
// @Success 201 {object} User
// @Failure 400 {object} zendia.APIError
// @Router /users [post]
func createUser(c *zendia.Context[CreateUserRequest]) error {
	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	// Simula criação do usuário
	user := User{
		ID:    3,
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	c.Created(user)
	return nil
}

// @Summary Update user
// @Description Update an existing user
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body CreateUserRequest true "User data"
// @Success 200 {object} User
// @Failure 400 {object} zendia.APIError
// @Failure 404 {object} zendia.APIError
// @Router /users/{id} [put]
func updateUser(c *zendia.Context[CreateUserRequest]) error {
	id := c.Param("id")
	if id != "1" {
		return zendia.NewNotFoundError("User not found")
	}

	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	user := User{
		ID:    1,
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	c.Success(user)
	return nil
}

// @Summary Delete user
// @Description Delete a user by ID
// @Tags users
// @Param id path int true "User ID"
// @Success 204
// @Failure 404 {object} zendia.APIError
// @Router /users/{id} [delete]
func deleteUser(c *zendia.Context[any]) error {
	id := c.Param("id")
	if id != "1" {
		return zendia.NewNotFoundError("User not found")
	}

	c.NoContent()
	return nil
}

// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func healthCheck(c *zendia.Context[any]) error {
	c.Success(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
	})
	return nil
}