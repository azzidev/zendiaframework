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
func (z *Zendia) SetupSwagger(info SwaggerInfo) {
	// HTML customizado sem navbar
	z.engine.GET("/swagger/*any", func(c *gin.Context) {
		if c.Request.URL.Path == "/swagger/index.html" || c.Request.URL.Path == "/swagger/" {
			c.Header("Content-Type", "text/html")
			c.String(200, customSwaggerHTML)
			return
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	})
}

const customSwaggerHTML = `<!DOCTYPE html>
<html>
<head>
  <title>ZendiaTask API</title>
  <link rel="stylesheet" type="text/css" href="./swagger-ui-bundle.css" />
  <style>
    .topbar { display: none !important; }
    .swagger-ui .topbar { display: none !important; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="./swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: './doc.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.presets.standalone
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

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