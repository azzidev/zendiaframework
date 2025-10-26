package zendia

import (
	"context"
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
		ResponseMessage: message,
		ResponseSuccess: true,
		ResponseData:    data,
	})
}

// Created retorna uma resposta de criação bem-sucedida
func (c *Context[T]) Created(message string, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		ResponseSuccess: true,
		ResponseMessage: message,
		ResponseData:    data,
	})
}

// Updated retorna uma resposta de atualização bem-sucedida
func (c *Context[T]) Updated(message string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		ResponseSuccess: true,
		ResponseMessage: message,
		ResponseData:    data,
	})
}

// NoContent retorna uma resposta sem conteúdo
func (c *Context[T]) NoContent() {
	c.Status(http.StatusNoContent)
}

// Fail retorna uma resposta de erro padronizada
func (c *Context[T]) Fail(code int, message string, err error) {
	response := gin.H{
		ResponseSuccess: false,
		ResponseMessage: message,
	}
	if err != nil {
		response[ResponseError] = err.Error()
	}
	c.JSON(code, response)
}

// BadRequest retorna um erro de requisição inválida
func (c *Context[T]) BadRequest(message string) {
	c.Fail(http.StatusBadRequest, message, nil)
}

// BadRequestWithError retorna um erro de requisição inválida com erro detalhado
func (c *Context[T]) BadRequestWithError(message string, err error) {
	c.Fail(http.StatusBadRequest, message, err)
}

// NotFound retorna um erro de recurso não encontrado
func (c *Context[T]) NotFound(message string) {
	c.Fail(http.StatusNotFound, message, nil)
}

// NotFoundWithError retorna um erro de recurso não encontrado com erro detalhado
func (c *Context[T]) NotFoundWithError(message string, err error) {
	c.Fail(http.StatusNotFound, message, err)
}

// InternalError retorna um erro interno do servidor
func (c *Context[T]) InternalError(message string) {
	c.Fail(http.StatusInternalServerError, message, nil)
}

// InternalErrorWithError retorna um erro interno com erro detalhado
func (c *Context[T]) InternalErrorWithError(message string, err error) {
	c.Fail(http.StatusInternalServerError, message, err)
}

// Conflict retorna um erro de conflito (409)
func (c *Context[T]) Conflict(message string) {
	c.Fail(http.StatusConflict, message, nil)
}

// ConflictWithError retorna um erro de conflito com erro detalhado
func (c *Context[T]) ConflictWithError(message string, err error) {
	c.Fail(http.StatusConflict, message, err)
}

// Unauthorized retorna um erro de não autorizado (401)
func (c *Context[T]) Unauthorized(message string) {
	c.Fail(http.StatusUnauthorized, message, nil)
}

// Forbidden retorna um erro de proibido (403)
func (c *Context[T]) Forbidden(message string) {
	c.Fail(http.StatusForbidden, message, nil)
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

// SetTenant seta o tenant ID no contexto da sessão
func (c *Context[T]) SetTenant(tenantID string) {
	c.Set(AuthTenantIDKey, tenantID)
	c.Set(TenantIDKey, tenantID)
	c.Header(HeaderTenantID, tenantID)
	
	// Atualiza context para auditoria
	ctx := context.WithValue(c.Request.Context(), ContextTenantID, tenantID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	c.Request = c.Request.WithContext(ctx)
}

// SetUserID seta o user ID customizado no contexto
func (c *Context[T]) SetUserID(userID string) {
	c.Set(AuthUserIDKey, userID)
	c.Set(UserIDKey, userID)
	c.Header(HeaderUserID, userID)
	
	// Atualiza context para auditoria
	ctx := context.WithValue(c.Request.Context(), ContextUserID, userID)
	ctx = context.WithValue(ctx, UserIDKey, userID)
	c.Request = c.Request.WithContext(ctx)
}

// SetRole seta a role do usuário no contexto
func (c *Context[T]) SetRole(role string) {
	c.Set(AuthRoleKey, role)
}

// SetUserName seta o nome do usuário no contexto
func (c *Context[T]) SetUserName(userName string) {
	c.Set(AuthNameKey, userName)
	c.Set(UserNameKey, userName)
	
	// Atualiza context
	ctx := context.WithValue(c.Request.Context(), UserNameKey, userName)
	c.Request = c.Request.WithContext(ctx)
}
