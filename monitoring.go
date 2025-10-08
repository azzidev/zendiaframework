package zendia

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics estrutura para métricas da aplicação
type Metrics struct {
	mu              sync.RWMutex
	RequestCount    map[string]int64     `json:"request_count"`
	ResponseTimes   map[string][]float64 `json:"response_times"`
	ErrorCount      map[string]int64     `json:"error_count"`
	ActiveRequests  int64                `json:"active_requests"`
	StartTime       time.Time            `json:"start_time"`
}

// NewMetrics cria uma nova instância de métricas
func NewMetrics() *Metrics {
	return &Metrics{
		RequestCount:   make(map[string]int64),
		ResponseTimes:  make(map[string][]float64),
		ErrorCount:     make(map[string]int64),
		ActiveRequests: 0,
		StartTime:      time.Now(),
	}
}

// RecordRequest registra uma requisição
func (m *Metrics) RecordRequest(method, path string, duration time.Duration, statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s %s", method, path)
	m.RequestCount[key]++
	m.ResponseTimes[key] = append(m.ResponseTimes[key], duration.Seconds())
	
	if statusCode >= 400 {
		m.ErrorCount[key]++
	}
}

// IncrementActive incrementa requisições ativas
func (m *Metrics) IncrementActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveRequests++
}

// DecrementActive decrementa requisições ativas
func (m *Metrics) DecrementActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveRequests--
}

// GetStats retorna estatísticas das métricas
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"uptime":          time.Since(m.StartTime).String(),
		"active_requests": m.ActiveRequests,
		"total_requests":  m.getTotalRequests(),
		"total_errors":    m.getTotalErrors(),
		"endpoints":       m.getEndpointStats(),
	}
	
	return stats
}

func (m *Metrics) getTotalRequests() int64 {
	var total int64
	for _, count := range m.RequestCount {
		total += count
	}
	return total
}

func (m *Metrics) getTotalErrors() int64 {
	var total int64
	for _, count := range m.ErrorCount {
		total += count
	}
	return total
}

func (m *Metrics) getEndpointStats() map[string]interface{} {
	endpoints := make(map[string]interface{})
	
	for endpoint, count := range m.RequestCount {
		avgTime := 0.0
		if times, exists := m.ResponseTimes[endpoint]; exists && len(times) > 0 {
			var sum float64
			for _, t := range times {
				sum += t
			}
			avgTime = sum / float64(len(times))
		}
		
		endpoints[endpoint] = map[string]interface{}{
			"requests":     count,
			"errors":       m.ErrorCount[endpoint],
			"avg_time_ms":  avgTime * 1000,
		}
	}
	
	return endpoints
}

// Monitoring middleware para coleta de métricas
func Monitoring(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		metrics.IncrementActive()
		
		c.Next()
		
		duration := time.Since(start)
		metrics.DecrementActive()
		metrics.RecordRequest(c.Request.Method, c.FullPath(), duration, c.Writer.Status())
	}
}

// AddMonitoring adiciona middleware de monitoramento ao Zendia
func (z *Zendia) AddMonitoring() *Metrics {
	metrics := NewMetrics()
	z.Use(Monitoring(metrics))
	return metrics
}