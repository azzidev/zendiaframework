package zendia

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Framework representa a instância principal do ZendiaFramework
type Framework struct {
	engine      *gin.Engine
	middlewares []gin.HandlerFunc
	validator   *Validator
	errorHandler ErrorHandler
}

// New cria uma nova instância do framework
func New() *Framework {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	
	f := &Framework{
		engine:       engine,
		middlewares:  []gin.HandlerFunc{},
		validator:    NewValidator(),
		errorHandler: NewErrorHandler(),
	}
	
	// Middleware padrão de recuperação de panic
	f.engine.Use(gin.Recovery())
	
	return f
}

// Use adiciona middleware global
func (f *Framework) Use(middleware ...gin.HandlerFunc) {
	f.middlewares = append(f.middlewares, middleware...)
	f.engine.Use(middleware...)
}

// Group cria um grupo de rotas
func (f *Framework) Group(relativePath string, handlers ...gin.HandlerFunc) *RouteGroup {
	ginGroup := f.engine.Group(relativePath, handlers...)
	return &RouteGroup{
		group:     ginGroup,
		framework: f,
	}
}

// GET registra uma rota GET
func (f *Framework) GET(relativePath string, handlers ...gin.HandlerFunc) {
	f.engine.GET(relativePath, handlers...)
}

// POST registra uma rota POST
func (f *Framework) POST(relativePath string, handlers ...gin.HandlerFunc) {
	f.engine.POST(relativePath, handlers...)
}

// PUT registra uma rota PUT
func (f *Framework) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	f.engine.PUT(relativePath, handlers...)
}

// DELETE registra uma rota DELETE
func (f *Framework) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	f.engine.DELETE(relativePath, handlers...)
}

// PATCH registra uma rota PATCH
func (f *Framework) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	f.engine.PATCH(relativePath, handlers...)
}

// Run inicia o servidor
func (f *Framework) Run(addr ...string) error {
	return f.engine.Run(addr...)
}

// ServeHTTP implementa http.Handler
func (f *Framework) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f.engine.ServeHTTP(w, req)
}

// GetValidator retorna o validador
func (f *Framework) GetValidator() *Validator {
	return f.validator
}

// GetErrorHandler retorna o manipulador de erros
func (f *Framework) GetErrorHandler() ErrorHandler {
	return f.errorHandler
}