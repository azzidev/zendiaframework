package zendia

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Context é um wrapper do gin.Context com funcionalidades adicionais
type Context[T any] struct {
	*gin.Context
}

// BindJSON faz o bind e validação de dados JSON
func (c *Context[T]) BindJSON(obj *T) error {
	if err := c.Context.ShouldBindJSON(obj); err != nil {
		return NewValidationError("Invalid JSON data", err)
	}

	// Valida usando o validator customizado
	validator := NewValidator()
	if err := validator.Validate(obj); err != nil {
		return err
	}

	return nil
}

// BindQuery faz o bind e validação de query parameters
func (c *Context[T]) BindQuery(obj *T) error {
	if err := c.Context.ShouldBindQuery(obj); err != nil {
		return NewValidationError("Invalid query parameters", err)
	}

	// Valida usando o validator customizado
	validator := NewValidator()
	if err := validator.Validate(obj); err != nil {
		return err
	}

	return nil
}

// BindURI faz o bind e validação de parâmetros da URI
func (c *Context[T]) BindURI(obj *T) error {
	if err := c.Context.ShouldBindUri(obj); err != nil {
		return NewValidationError("Invalid URI parameters", err)
	}

	// Valida usando o validator customizado
	validator := NewValidator()
	if err := validator.Validate(obj); err != nil {
		return err
	}

	return nil
}

// Success retorna uma resposta de sucesso padronizada
func (c *Context[T]) Success(message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"success": true,
		"data":    data,
	})
}

// Created retorna uma resposta de criação bem-sucedida
func (c *Context[T]) Created(message string, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// NoContent retorna uma resposta sem conteúdo
func (c *Context[T]) NoContent() {
	c.Status(http.StatusNoContent)
}

// BadRequest retorna um erro de requisição inválida
func (c *Context[T]) BadRequest(message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   message,
	})
}

// NotFound retorna um erro de recurso não encontrado
func (c *Context[T]) NotFound(message string) {
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   message,
	})
}

// InternalError retorna um erro interno do servidor
func (c *Context[T]) InternalError(message string) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   message,
	})
}

// GetTenantID retorna o tenant ID do contexto
func (c *Context[T]) GetTenantID() string {
	return GetTenantIDFromGin(c.Context)
}

// GetUserID retorna o user ID do contexto
func (c *Context[T]) GetUserID() string {
	return GetUserIDFromGin(c.Context)
}

// GetActionAt retorna o timestamp da ação
func (c *Context[T]) GetActionAt() time.Time {
	return GetActionAtFromGin(c.Context)
}

// GetTenantInfo retorna todas as informações do tenant
func (c *Context[T]) GetTenantInfo() TenantInfo {
	return GetTenantInfoFromGin(c.Context)
}
