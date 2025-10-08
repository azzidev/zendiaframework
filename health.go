package zendia

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// HealthStatus representa o status de saúde
type HealthStatus string

const (
	HealthStatusUp   HealthStatus = "UP"
	HealthStatusDown HealthStatus = "DOWN"
	HealthStatusWarn HealthStatus = "WARN"
)

// HealthCheck interface para verificações de saúde
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthCheckResult
}

// HealthCheckResult resultado de uma verificação
type HealthCheckResult struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Details interface{}  `json:"details,omitempty"`
}

// HealthManager gerencia verificações de saúde
type HealthManager struct {
	mu     sync.RWMutex
	checks map[string]HealthCheck
}

// NewHealthManager cria um novo gerenciador de saúde
func NewHealthManager() *HealthManager {
	return &HealthManager{
		checks: make(map[string]HealthCheck),
	}
}

// AddCheck adiciona uma verificação de saúde
func (hm *HealthManager) AddCheck(check HealthCheck) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checks[check.Name()] = check
}

// RemoveCheck remove uma verificação de saúde
func (hm *HealthManager) RemoveCheck(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.checks, name)
}

// CheckHealth executa todas as verificações
func (hm *HealthManager) CheckHealth(ctx context.Context) map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	overallStatus := HealthStatusUp

	for name, check := range hm.checks {
		result := check.Check(ctx)
		results[name] = result

		if result.Status == HealthStatusDown {
			overallStatus = HealthStatusDown
		} else if result.Status == HealthStatusWarn && overallStatus == HealthStatusUp {
			overallStatus = HealthStatusWarn
		}
	}

	return map[string]interface{}{
		"status":    overallStatus,
		"checks":    results,
		"timestamp": time.Now(),
	}
}

// DatabaseHealthCheck verificação de saúde do banco de dados
type DatabaseHealthCheck struct {
	name string
	ping func(context.Context) error
}

// NewDatabaseHealthCheck cria verificação de BD
func NewDatabaseHealthCheck(name string, pingFunc func(context.Context) error) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		name: name,
		ping: pingFunc,
	}
}

func (d *DatabaseHealthCheck) Name() string {
	return d.name
}

func (d *DatabaseHealthCheck) Check(ctx context.Context) HealthCheckResult {
	start := time.Now()
	if err := d.ping(ctx); err != nil {
		return HealthCheckResult{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("Database connection failed: %v", err),
			Details: map[string]interface{}{
				"response_time_ms": time.Since(start).Milliseconds(),
				"error": err.Error(),
			},
		}
	}
	return HealthCheckResult{
		Status:  HealthStatusUp,
		Message: "Database connection successful",
		Details: map[string]interface{}{
			"response_time_ms": time.Since(start).Milliseconds(),
		},
	}
}

// MemoryHealthCheck verificação de uso de memória
type MemoryHealthCheck struct {
	maxMemoryMB int64
}

// NewMemoryHealthCheck cria verificação de memória
func NewMemoryHealthCheck(maxMemoryMB int64) *MemoryHealthCheck {
	return &MemoryHealthCheck{
		maxMemoryMB: maxMemoryMB,
	}
}

func (m *MemoryHealthCheck) Name() string {
	return "memory"
}

func (m *MemoryHealthCheck) Check(ctx context.Context) HealthCheckResult {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	currentMemoryMB := int64(memStats.Alloc / 1024 / 1024)
	heapMB := int64(memStats.HeapAlloc / 1024 / 1024)
	sysMB := int64(memStats.Sys / 1024 / 1024)
	
	details := map[string]interface{}{
		"alloc_mb":     currentMemoryMB,
		"heap_mb":      heapMB,
		"sys_mb":       sysMB,
		"max_mb":       m.maxMemoryMB,
		"gc_cycles":    memStats.NumGC,
		"goroutines":   runtime.NumGoroutine(),
	}
	
	if currentMemoryMB > m.maxMemoryMB {
		return HealthCheckResult{
			Status:  HealthStatusDown,
			Message: fmt.Sprintf("Memory usage critical: %dMB (max: %dMB)", currentMemoryMB, m.maxMemoryMB),
			Details: details,
		}
	}
	
	if currentMemoryMB > m.maxMemoryMB*80/100 {
		return HealthCheckResult{
			Status:  HealthStatusWarn,
			Message: fmt.Sprintf("Memory usage high: %dMB (80%% of max)", currentMemoryMB),
			Details: details,
		}
	}
	
	return HealthCheckResult{
		Status:  HealthStatusUp,
		Message: "Memory usage normal",
		Details: details,
	}
}

// AddHealthEndpoint adiciona endpoint de saúde ao grupo
func (rg *RouteGroup) AddHealthEndpoint(healthManager *HealthManager) {
	rg.GET("/health", Handle(func(c *Context[any]) error {
		ctx := context.Background()
		health := healthManager.CheckHealth(ctx)

		status := health["status"].(HealthStatus)
		if status == HealthStatusDown {
			c.JSON(503, health)
		} else {
			c.Success(health)
		}
		return nil
	}))
}

// AddHealthEndpoint adiciona endpoint de saúde ao Zendia principal
func (z *Zendia) AddHealthEndpoint(healthManager *HealthManager) {
	z.GET("/health", Handle(func(c *Context[any]) error {
		ctx := context.Background()
		health := healthManager.CheckHealth(ctx)

		status := health["status"].(HealthStatus)
		if status == HealthStatusDown {
			c.JSON(503, health)
		} else {
			c.Success(health)
		}
		return nil
	}))
}
