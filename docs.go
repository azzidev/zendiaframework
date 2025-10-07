package zendia

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerInfo contém informações da documentação
type SwaggerInfo struct {
	Title       string
	Description string
	Version     string
	Host        string
	BasePath    string
}

// SetupSwagger configura a documentação Swagger
func (f *Framework) SetupSwagger(info SwaggerInfo) {
	// Configuração básica do Swagger
	f.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// APIDoc representa uma anotação de documentação
type APIDoc struct {
	Summary     string
	Description string
	Tags        []string
	Accept      string
	Produce     string
	Param       []ParamDoc
	Success     ResponseDoc
	Failure     []ResponseDoc
}

// ParamDoc representa um parâmetro da API
type ParamDoc struct {
	Name        string
	In          string // query, path, header, body
	Type        string
	Required    bool
	Description string
}

// ResponseDoc representa uma resposta da API
type ResponseDoc struct {
	Code        int
	Description string
	Schema      interface{}
}

// Doc adiciona documentação a uma rota (placeholder para integração com swag)
func Doc(doc APIDoc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Esta função serve como marcador para a geração automática de docs
		// O swag irá processar os comentários acima das funções handler
		c.Next()
	}
}

// Example de como usar a documentação:
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body User true "User data"
// @Success 201 {object} User
// @Failure 400 {object} APIError
// @Router /users [post]