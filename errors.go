package zendia

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorType define tipos de erro
type ErrorType int

const (
	ValidationErrorType ErrorType = iota
	NotFoundErrorType
	UnauthorizedErrorType
	InternalErrorType
	BadRequestErrorType
)

// APIError representa um erro da API
type APIError struct {
	Type    ErrorType `json:"-"`
	Message string    `json:"message"`
	Details error     `json:"details,omitempty"`
	Code    int       `json:"code"`
}

func (e *APIError) Error() string {
	return e.Message
}

// ErrorHandler interface para manipulação de erros
type ErrorHandler interface {
	Handle(c *gin.Context, err error)
}

// DefaultErrorHandler implementação padrão do manipulador de erros
type DefaultErrorHandler struct{}

// NewErrorHandler cria um novo manipulador de erros
func NewErrorHandler() ErrorHandler {
	return &DefaultErrorHandler{}
}

// Handle processa erros e retorna respostas apropriadas
func (h *DefaultErrorHandler) Handle(c *gin.Context, err error) {
	if apiErr, ok := err.(*APIError); ok {
		c.JSON(apiErr.Code, gin.H{
			"success": false,
			"error":   apiErr.Message,
		})
		return
	}
	
	// Erro genérico
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   "Internal server error",
	})
}

// NewValidationError cria um erro de validação
func NewValidationError(message string, details error) *APIError {
	return &APIError{
		Type:    ValidationErrorType,
		Message: message,
		Details: details,
		Code:    http.StatusBadRequest,
	}
}

// NewNotFoundError cria um erro de recurso não encontrado
func NewNotFoundError(message string) *APIError {
	return &APIError{
		Type:    NotFoundErrorType,
		Message: message,
		Code:    http.StatusNotFound,
	}
}

// NewUnauthorizedError cria um erro de não autorizado
func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Type:    UnauthorizedErrorType,
		Message: message,
		Code:    http.StatusUnauthorized,
	}
}

// NewInternalError cria um erro interno
func NewInternalError(message string) *APIError {
	return &APIError{
		Type:    InternalErrorType,
		Message: message,
		Code:    http.StatusInternalServerError,
	}
}

// NewBadRequestError cria um erro de requisição inválida
func NewBadRequestError(message string) *APIError {
	return &APIError{
		Type:    BadRequestErrorType,
		Message: message,
		Code:    http.StatusBadRequest,
	}
}

// ErrorMiddleware middleware para captura e tratamento de erros
func ErrorMiddleware(handler ErrorHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handler.Handle(c, err)
		}
	}
}