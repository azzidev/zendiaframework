package zendia

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator encapsula o validador
type Validator struct {
	validate *validator.Validate
}

// NewValidator cria uma nova instância do validador
func NewValidator() *Validator {
	v := validator.New()
	
	// Registra função para obter nome do campo JSON
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	
	return &Validator{validate: v}
}

// Validate valida uma estrutura
func (v *Validator) Validate(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if len(validationErrors) == 1 {
			// Otimização: se há apenas um erro, não precisa de slice
			return NewValidationError("Validation failed", fmt.Errorf(v.formatError(validationErrors[0])))
		}
		
		// Para múltiplos erros, usa strings.Builder para melhor performance
		var builder strings.Builder
		for i, err := range validationErrors {
			if i > 0 {
				builder.WriteString("; ")
			}
			builder.WriteString(v.formatError(err))
		}
		return NewValidationError("Validation failed", fmt.Errorf(builder.String()))
	}
	return nil
}

// RegisterValidation registra uma validação customizada
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// Regex compilada uma vez para melhor performance
var controlCharsRegex = regexp.MustCompile(`[\r\n\t\x00-\x1f\x7f-\x9f]`)

// sanitizeLogValue prevents log injection by sanitizing values
func sanitizeLogValue(value string) string {
	// Limit length first to prevent DoS
	if len(value) > 100 {
		value = value[:100] + "..."
	}
	
	// Quick check: se não há caracteres de controle, retorna direto
	hasControlChars := false
	for _, r := range value {
		if r < 32 || (r >= 127 && r <= 159) {
			hasControlChars = true
			break
		}
	}
	
	if !hasControlChars {
		return value
	}
	
	// Remove control characters apenas se necessário
	return controlCharsRegex.ReplaceAllString(value, "")
}

// formatError formats validation errors in Portuguese with log injection protection
func (v *Validator) formatError(err validator.FieldError) string {
	// Sanitize field name and parameters to prevent log injection
	field := sanitizeLogValue(err.Field())
	tag := sanitizeLogValue(err.Tag())
	param := sanitizeLogValue(err.Param())
	
	switch tag {
	case "required":
		return fmt.Sprintf("%s é obrigatório", field)
	case "email":
		return fmt.Sprintf("%s deve ser um email válido", field)
	case "min":
		return fmt.Sprintf("%s deve ter pelo menos %s caracteres", field, param)
	case "max":
		return fmt.Sprintf("%s deve ter no máximo %s caracteres", field, param)
	case "len":
		return fmt.Sprintf("%s deve ter exatamente %s caracteres", field, param)
	case "gt":
		return fmt.Sprintf("%s deve ser maior que %s", field, param)
	case "gte":
		return fmt.Sprintf("%s deve ser maior ou igual a %s", field, param)
	case "lt":
		return fmt.Sprintf("%s deve ser menor que %s", field, param)
	case "lte":
		return fmt.Sprintf("%s deve ser menor ou igual a %s", field, param)
	case "oneof":
		return fmt.Sprintf("%s deve ser um dos valores: %s", field, param)
	case "uuid":
		return fmt.Sprintf("%s deve ser um UUID válido", field)
	case "numeric":
		return fmt.Sprintf("%s deve ser numérico", field)
	case "alpha":
		return fmt.Sprintf("%s deve conter apenas letras", field)
	case "alphanum":
		return fmt.Sprintf("%s deve conter apenas letras e números", field)
	default:
		return fmt.Sprintf("%s falhou na validação '%s'", field, tag)
	}
}

