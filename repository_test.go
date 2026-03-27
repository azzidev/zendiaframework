package zendia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputSanitizer(t *testing.T) {
	sanitizer := NewInputSanitizer("project_id", "sprint_id")

	// Campos permitidos
	input := map[string]interface{}{
		"name":       "João",
		"email":      "joao@test.com",
		"project_id": "123",
	}

	result, err := sanitizer.Sanitize(input)
	assert.NoError(t, err)
	assert.Equal(t, "João", result["name"])
	assert.Equal(t, "joao@test.com", result["email"])
	assert.Equal(t, "123", result["project_id"])

	// Campos perigosos rejeitados
	dangerous := map[string]interface{}{
		"$where":   "function() { return true }",
		"name":     "safe",
	}

	result, err = sanitizer.Sanitize(dangerous)
	assert.NoError(t, err)
	assert.Equal(t, "safe", result["name"])
	_, exists := result["$where"]
	assert.False(t, exists)

	// Valores perigosos rejeitados
	dangerousValues := map[string]interface{}{
		"name": "$ne",
	}

	result, err = sanitizer.Sanitize(dangerousValues)
	assert.NoError(t, err)
	_, exists = result["name"]
	assert.False(t, exists)
}

func TestInputSanitizer_TooManyFields(t *testing.T) {
	sanitizer := NewInputSanitizer()

	input := make(map[string]interface{})
	for i := 0; i < 25; i++ {
		input["field"+string(rune('a'+i))] = "value"
	}

	_, err := sanitizer.Sanitize(input)
	assert.Error(t, err)
}
