package zendia

import "github.com/gin-gonic/gin"

// RouteGroup representa um grupo de rotas
type RouteGroup struct {
	group     *gin.RouterGroup
	framework *Framework
}

// Group cria um subgrupo de rotas
func (rg *RouteGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *RouteGroup {
	subGroup := rg.group.Group(relativePath, handlers...)
	return &RouteGroup{
		group:     subGroup,
		framework: rg.framework,
	}
}

// GET registra uma rota GET no grupo
func (rg *RouteGroup) GET(relativePath string, handlers ...gin.HandlerFunc) {
	rg.group.GET(relativePath, handlers...)
}

// POST registra uma rota POST no grupo
func (rg *RouteGroup) POST(relativePath string, handlers ...gin.HandlerFunc) {
	rg.group.POST(relativePath, handlers...)
}

// PUT registra uma rota PUT no grupo
func (rg *RouteGroup) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	rg.group.PUT(relativePath, handlers...)
}

// DELETE registra uma rota DELETE no grupo
func (rg *RouteGroup) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	rg.group.DELETE(relativePath, handlers...)
}

// PATCH registra uma rota PATCH no grupo
func (rg *RouteGroup) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	rg.group.PATCH(relativePath, handlers...)
}

// Use adiciona middleware ao grupo
func (rg *RouteGroup) Use(middleware ...gin.HandlerFunc) {
	rg.group.Use(middleware...)
}

// Handler é uma função genérica para manipular requisições
type Handler[T any] func(*Context[T]) error

// Handle converte um Handler genérico para gin.HandlerFunc
func Handle[T any](handler Handler[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &Context[T]{Context: c}
		if err := handler(ctx); err != nil {
			ctx.Error(err)
		}
	}
}