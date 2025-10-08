package zendia

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPHealthCheck verifica saúde de serviços HTTP
type HTTPHealthCheck struct {
	name string
	url  string
	timeout time.Duration
}

// NewHTTPHealthCheck cria verificação HTTP
func NewHTTPHealthCheck(name, url string, timeout time.Duration) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		name: name,
		url:  url,
		timeout: timeout,
	}
}

func (h *HTTPHealthCheck) Name() string {
	return h.name
}

func (h *HTTPHealthCheck) Check(ctx context.Context) HealthCheckResult {
	client := &http.Client{Timeout: h.timeout}
	start := time.Now()
	
	resp, err := client.Get(h.url)
	responseTime := time.Since(start)
	
	if err != nil {
		return HealthCheckResult{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
			Details: map[string]interface{}{
				"url": h.url,
				"response_time_ms": responseTime.Milliseconds(),
				"error": err.Error(),
			},
		}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return HealthCheckResult{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("HTTP status %d", resp.StatusCode),
			Details: map[string]interface{}{
				"url": h.url,
				"status_code": resp.StatusCode,
				"response_time_ms": responseTime.Milliseconds(),
			},
		}
	}
	
	return HealthCheckResult{
		Status:  HealthStatusUp,
		Message: "HTTP service healthy",
		Details: map[string]interface{}{
			"url": h.url,
			"status_code": resp.StatusCode,
			"response_time_ms": responseTime.Milliseconds(),
		},
	}
}



// RepositoryHealthCheck verifica saúde do repository
type RepositoryHealthCheck struct {
	name string
	repo interface{}
}

// NewRepositoryHealthCheck cria verificação de repository
func NewRepositoryHealthCheck(name string, repo interface{}) *RepositoryHealthCheck {
	return &RepositoryHealthCheck{
		name: name,
		repo: repo,
	}
}

func (r *RepositoryHealthCheck) Name() string {
	return r.name
}

func (r *RepositoryHealthCheck) Check(ctx context.Context) HealthCheckResult {
	start := time.Now()
	
	// Tenta usar interface assertion para chamar métodos comuns
	if mongoRepo, ok := r.repo.(interface{ GetAllSkipTake(context.Context, map[string]interface{}, int, int) (interface{}, error) }); ok {
		_, err := mongoRepo.GetAllSkipTake(ctx, map[string]interface{}{}, 0, 1)
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusDown,
				Message: fmt.Sprintf("Repository check failed: %v", err),
				Details: map[string]interface{}{
					"type": "repository",
					"response_time_ms": time.Since(start).Milliseconds(),
					"error": err.Error(),
				},
			}
		}
	} else if memRepo, ok := r.repo.(interface{ GetAll(context.Context, map[string]interface{}) (interface{}, error) }); ok {
		_, err := memRepo.GetAll(ctx, map[string]interface{}{})
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusDown,
				Message: fmt.Sprintf("Repository check failed: %v", err),
				Details: map[string]interface{}{
					"type": "repository",
					"response_time_ms": time.Since(start).Milliseconds(),
					"error": err.Error(),
				},
			}
		}
	} else {
		return HealthCheckResult{
			Status:  HealthStatusWarn,
			Message: "Repository type not supported for health check",
			Details: map[string]interface{}{
				"type": "unknown",
			},
		}
	}

	return HealthCheckResult{
		Status:  HealthStatusUp,
		Message: "Repository healthy",
		Details: map[string]interface{}{
			"response_time_ms": time.Since(start).Milliseconds(),
		},
	}
}

