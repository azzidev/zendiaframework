package zendia

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Zendia representa a instância principal do ZendiaFramework
type Zendia struct {
	engine      *gin.Engine
	middlewares []gin.HandlerFunc
	validator   *Validator
	errorHandler ErrorHandler
}

// New cria uma nova instância do framework
func New() *Zendia {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	
	z := &Zendia{
		engine:       engine,
		middlewares:  []gin.HandlerFunc{},
		validator:    NewValidator(),
		errorHandler: NewErrorHandler(),
	}
	
	// Middlewares padrão
	z.engine.Use(gin.Recovery())
	z.engine.Use(TenantMiddleware(nil)) // Usa extrator padrão
	
	return z
}

// Use adiciona middleware global
func (z *Zendia) Use(middleware ...gin.HandlerFunc) {
	z.middlewares = append(z.middlewares, middleware...)
	z.engine.Use(middleware...)
}

// Group cria um grupo de rotas
func (z *Zendia) Group(relativePath string, handlers ...gin.HandlerFunc) *RouteGroup {
	ginGroup := z.engine.Group(relativePath, handlers...)
	return &RouteGroup{
		group:     ginGroup,
		zendia: z,
	}
}

// GET registra uma rota GET
func (z *Zendia) GET(relativePath string, handlers ...gin.HandlerFunc) {
	z.engine.GET(relativePath, handlers...)
}

// POST registra uma rota POST
func (z *Zendia) POST(relativePath string, handlers ...gin.HandlerFunc) {
	z.engine.POST(relativePath, handlers...)
}

// PUT registra uma rota PUT
func (z *Zendia) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	z.engine.PUT(relativePath, handlers...)
}

// DELETE registra uma rota DELETE
func (z *Zendia) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	z.engine.DELETE(relativePath, handlers...)
}

// PATCH registra uma rota PATCH
func (z *Zendia) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	z.engine.PATCH(relativePath, handlers...)
}

// Run inicia o servidor
func (z *Zendia) Run(addr ...string) error {
	return z.engine.Run(addr...)
}

// ServeHTTP implementa http.Handler
func (z *Zendia) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	z.engine.ServeHTTP(w, req)
}

// GetValidator retorna o validador
func (z *Zendia) GetValidator() *Validator {
	return z.validator
}

// GetErrorHandler retorna o manipulador de erros
func (z *Zendia) GetErrorHandler() ErrorHandler {
	return z.errorHandler
}

// SetTenantExtractor configura um extrator customizado de tenant
func (z *Zendia) SetTenantExtractor(extractor TenantExtractor) {
	// Remove o middleware padrão e adiciona o customizado
	z.Use(TenantMiddleware(extractor))
}