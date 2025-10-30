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
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, v.formatError(err))
		}
		return NewValidationError("Validation failed", fmt.Errorf(strings.Join(errors, "; ")))
	}
	return nil
}

// RegisterValidation registra uma validação customizada
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// sanitizeLogValue prevents log injection by sanitizing values
func sanitizeLogValue(value string) string {
	// Remove control characters and newlines to prevent log injection
	re := regexp.MustCompile(`[\r\n\t\x00-\x1f\x7f-\x9f]`)
	value = re.ReplaceAllString(value, "")
	// Limit length to prevent DoS
	if len(value) > 100 {
		value = value[:100] + "..."
	}
	return value
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

// ValidateStruct é um helper para validar estruturas
func ValidateStruct[T any](v *Validator, data T) error {
	return v.Validate(data)
}