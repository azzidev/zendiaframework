package zendia

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFramework_New(t *testing.T) {
	app := New()
	assert.NotNil(t, app)
	assert.NotNil(t, app.engine)
	assert.NotNil(t, app.validator)
	assert.NotNil(t, app.errorHandler)
}

func TestFramework_GET(t *testing.T) {
	app := New()
	
	app.GET("/test", Handle(func(c *Context[any]) error {
		c.Success("test response")
		return nil
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "test response", response["data"])
}

func TestFramework_POST(t *testing.T) {
	app := New()
	
	type TestRequest struct {
		Name string `json:"name" validate:"required"`
	}
	
	app.POST("/test", Handle(func(c *Context[TestRequest]) error {
		var req TestRequest
		if err := c.BindJSON(&req); err != nil {
			return err
		}
		c.Created(req)
		return nil
	}))

	testData := TestRequest{Name: "test"}
	jsonData, _ := json.Marshal(testData)
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestFramework_Group(t *testing.T) {
	app := New()
	
	api := app.Group("/api")
	api.GET("/test", Handle(func(c *Context[any]) error {
		c.Success("group test")
		return nil
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()
	
	type TestStruct struct {
		Name  string `validate:"required,min=2"`
		Email string `validate:"required,email"`
	}
	
	// Teste com dados válidos
	valid := TestStruct{Name: "João", Email: "joao@example.com"}
	err := validator.Validate(valid)
	assert.NoError(t, err)
	
	// Teste com dados inválidos
	invalid := TestStruct{Name: "A", Email: "invalid-email"}
	err = validator.Validate(invalid)
	assert.Error(t, err)
}

func TestErrorHandler_Handle(t *testing.T) {
	app := New()
	
	app.GET("/error", Handle(func(c *Context[any]) error {
		return NewNotFoundError("Resource not found")
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	app.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "Resource not found", response["error"])
}

func TestMiddleware_CORS(t *testing.T) {
	app := New()
	app.Use(CORS())
	
	app.GET("/test", Handle(func(c *Context[any]) error {
		c.Success("test")
		return nil
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	app.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestMiddleware_Auth(t *testing.T) {
	app := New()
	
	authMiddleware := Auth(func(token string) bool {
		return token == "valid-token"
	})
	
	app.GET("/protected", authMiddleware, Handle(func(c *Context[any]) error {
		c.Success("protected resource")
		return nil
	}))

	// Teste sem token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Teste com token válido
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	app.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}