package zendia

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// MetricsConfig configuração para métricas
type MetricsConfig struct {
	MaxEndpoints      int           // Máximo de endpoints para rastrear
	MaxResponseTimes  int           // Máximo de response times por endpoint
	CleanupInterval   time.Duration // Intervalo de limpeza automática
	MaxMemoryMB      int64         // Máximo de memória em MB
	PersistInterval   time.Duration // Intervalo para salvar no banco
	EnablePersistence bool          // Se deve salvar no banco
}

// DefaultMetricsConfig configuração padrão segura
var DefaultMetricsConfig = MetricsConfig{
	MaxEndpoints:      100,
	MaxResponseTimes:  1000,
	CleanupInterval:   5 * time.Minute,
	MaxMemoryMB:      10, // 10MB max
	PersistInterval:   2 * time.Minute, // Salva a cada 2 minutos (menos agressivo)
	EnablePersistence: false, // Desabilitado por padrão para evitar crashes
}

// EndpointStats estatísticas por endpoint
type EndpointStats struct {
	Requests      int64     `json:"requests"`
	Errors        int64     `json:"errors"`
	TotalTime     float64   `json:"-"` // Para calcular média
	LastAccess    time.Time `json:"-"` // Para limpeza
}

// MetricsSnapshot snapshot das métricas para persistência
type MetricsSnapshot struct {
	ID             string                 `bson:"_id" json:"id"`
	Timestamp      time.Time              `bson:"timestamp" json:"timestamp"`
	TenantID       string                 `bson:"tenant_id" json:"tenant_id,omitempty"`
	Uptime         string                 `bson:"uptime" json:"uptime"`
	ActiveRequests int64                  `bson:"active_requests" json:"active_requests"`
	TotalRequests  int64                  `bson:"total_requests" json:"total_requests"`
	TotalErrors    int64                  `bson:"total_errors" json:"total_errors"`
	ErrorRate      float64                `bson:"error_rate" json:"error_rate"`
	Endpoints      map[string]interface{} `bson:"endpoints" json:"endpoints"`
	MemoryUsage    map[string]interface{} `bson:"memory_usage" json:"memory_usage"`
}

// MetricsPersister interface para persistência de métricas
type MetricsPersister interface {
	Save(snapshot MetricsSnapshot) error
	GetHistory(tenantID string, from, to time.Time) ([]MetricsSnapshot, error)
}

// Metrics estrutura para métricas da aplicação
type Metrics struct {
	mu             sync.RWMutex
	config         MetricsConfig
	stats          map[string]*EndpointStats `json:"endpoints"`
	ActiveRequests int64                     `json:"active_requests"`
	StartTime      time.Time                 `json:"start_time"`
	lastCleanup    time.Time
	lastPersist    time.Time
	persister      MetricsPersister
}

// NewMetrics cria uma nova instância de métricas
func NewMetrics() *Metrics {
	return NewMetricsWithConfig(DefaultMetricsConfig)
}

// NewMetricsWithConfig cria métricas com configuração customizada
func NewMetricsWithConfig(config MetricsConfig) *Metrics {
	m := &Metrics{
		config:      config,
		stats:       make(map[string]*EndpointStats),
		StartTime:   time.Now(),
		lastCleanup: time.Now(),
		lastPersist: time.Now(),
	}
	
	// Inicia limpeza automática
	go m.startCleanupRoutine()
	
	// NÃO inicia persistência automaticamente aqui
	// Será iniciada apenas quando SetPersister for chamado
	
	return m
}

// SetPersister configura o persistidor de métricas
func (m *Metrics) SetPersister(persister MetricsPersister) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persister = persister
}

// DisablePersistence desabilita a persistência de métricas
func (m *Metrics) DisablePersistence() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.EnablePersistence = false
	m.persister = nil
}

// EnablePersistence habilita a persistência de métricas
func (m *Metrics) EnablePersistence() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.EnablePersistence = true
}

// RecordRequest registra uma requisição com limites de segurança
func (m *Metrics) RecordRequest(method, path string, duration time.Duration, statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Verifica limite de endpoints
	key := fmt.Sprintf("%s %s", method, path)
	if len(m.stats) >= m.config.MaxEndpoints {
		if _, exists := m.stats[key]; !exists {
			return // Ignora novos endpoints se atingiu o limite
		}
	}
	
	// Cria ou atualiza stats
	if m.stats[key] == nil {
		m.stats[key] = &EndpointStats{}
	}
	
	stats := m.stats[key]
	stats.Requests++
	stats.TotalTime += duration.Seconds()
	stats.LastAccess = time.Now()
	
	if statusCode >= 400 {
		stats.Errors++
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
	
	totalReqs := m.getTotalRequests()
	totalErrs := m.getTotalErrors()
	errorRate := 0.0
	if totalReqs > 0 {
		errorRate = float64(totalErrs) / float64(totalReqs) * 100
	}
	
	stats := map[string]interface{}{
		"uptime":          time.Since(m.StartTime).String(),
		"active_requests": m.ActiveRequests,
		"total_requests":  totalReqs,
		"total_errors":    totalErrs,
		"error_rate":      errorRate,
		"endpoints":       m.getEndpointStats(),
		"memory":          m.GetMemoryUsage(),
		"persistence": map[string]interface{}{
			"enabled":      m.config.EnablePersistence,
			"interval":     m.config.PersistInterval.String(),
			"last_persist": m.lastPersist.Format(time.RFC3339),
		},
		"config": map[string]interface{}{
			"max_endpoints":     m.config.MaxEndpoints,
			"cleanup_interval": m.config.CleanupInterval.String(),
			"max_memory_mb":    m.config.MaxMemoryMB,
		},
	}
	
	return stats
}

func (m *Metrics) getTotalRequests() int64 {
	var total int64
	for _, stats := range m.stats {
		total += stats.Requests
	}
	return total
}

func (m *Metrics) getTotalErrors() int64 {
	var total int64
	for _, stats := range m.stats {
		total += stats.Errors
	}
	return total
}

// startCleanupRoutine inicia rotina de limpeza automática
func (m *Metrics) startCleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Metrics cleanup panic: %v\n", r)
					}
				}()
				m.cleanup()
			}()
		}
	}
}

// startPersistenceRoutine inicia rotina de persistência automática
func (m *Metrics) startPersistenceRoutine() {
	ticker := time.NewTicker(m.config.PersistInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Metrics persistence panic recovered: %v\n", r)
						// Não mata a aplicação, apenas loga o erro
					}
				}()
				
				// Verifica se persister está disponível antes de tentar persistir
				m.mu.RLock()
				hasPersister := m.persister != nil
				m.mu.RUnlock()
				
				if hasPersister {
					m.persistMetrics()
				}
			}()
		}
		}
}

// persistMetrics salva snapshot atual das métricas
func (m *Metrics) persistMetrics() {
	m.mu.RLock()
	persister := m.persister
	m.mu.RUnlock()
	
	if persister == nil {
		return
	}
	
	// Timeout para evitar travamento
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("persist panic: %v", r)
			}
		}()
		
		// Captura stats com tratamento de erro
		stats := m.GetStats()
		
		// Validação segura dos tipos
		uptime, _ := stats["uptime"].(string)
		activeReqs, _ := stats["active_requests"].(int64)
		totalReqs, _ := stats["total_requests"].(int64)
		totalErrs, _ := stats["total_errors"].(int64)
		errorRate, _ := stats["error_rate"].(float64)
		endpoints, _ := stats["endpoints"].(map[string]interface{})
		memory, _ := stats["memory"].(map[string]interface{})
		
		// Garante que endpoints e memory não sejam nil
		if endpoints == nil {
			endpoints = make(map[string]interface{})
		}
		if memory == nil {
			memory = make(map[string]interface{})
		}
		
		snapshot := MetricsSnapshot{
			ID:             fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp:      time.Now(),
			Uptime:         uptime,
			ActiveRequests: activeReqs,
			TotalRequests:  totalReqs,
			TotalErrors:    totalErrs,
			ErrorRate:      errorRate,
			Endpoints:      endpoints,
			MemoryUsage:    memory,
		}
		
		done <- persister.Save(snapshot)
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("Failed to persist metrics: %v\n", err)
		} else {
			m.mu.Lock()
			m.lastPersist = time.Now()
			m.mu.Unlock()
		}
	case <-time.After(10 * time.Second):
		fmt.Printf("Metrics persistence timeout after 10s\n")
	}
}

// GetMetricsHistory retorna histórico de métricas
func (m *Metrics) GetMetricsHistory(tenantID string, from, to time.Time) ([]MetricsSnapshot, error) {
	m.mu.RLock()
	persister := m.persister
	m.mu.RUnlock()
	
	if persister == nil {
		return nil, fmt.Errorf("persistence not configured")
	}
	
	return persister.GetHistory(tenantID, from, to)
}

// cleanup remove dados antigos para evitar memory leak
func (m *Metrics) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-m.config.CleanupInterval * 2) // Remove dados > 2x intervalo
	
	// Remove endpoints não acessados recentemente
	for endpoint, stats := range m.stats {
		if stats.LastAccess.Before(cutoff) {
			delete(m.stats, endpoint)
		}
	}
	
	m.lastCleanup = now
}

// GetMemoryUsage retorna uso aproximado de memória
func (m *Metrics) GetMemoryUsage() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Estimativa aproximada
	endpointCount := len(m.stats)
	estimatedMB := float64(endpointCount * 200) / 1024 / 1024 // ~200 bytes por endpoint
	
	return map[string]interface{}{
		"endpoints_tracked": endpointCount,
		"max_endpoints":    m.config.MaxEndpoints,
		"estimated_mb":     estimatedMB,
		"max_mb":           m.config.MaxMemoryMB,
		"last_cleanup":     m.lastCleanup.Format(time.RFC3339),
	}
}

func (m *Metrics) getEndpointStats() map[string]interface{} {
	endpoints := make(map[string]interface{})
	
	for endpoint, stats := range m.stats {
		avgTime := 0.0
		if stats.Requests > 0 {
			avgTime = stats.TotalTime / float64(stats.Requests)
		}
		
		endpoints[endpoint] = map[string]interface{}{
			"requests":     stats.Requests,
			"errors":       stats.Errors,
			"avg_time_ms":  avgTime * 1000,
			"error_rate":   float64(stats.Errors) / float64(stats.Requests) * 100,
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

// AddMonitoringWithPersistence adiciona monitoramento com persistência MongoDB
func (z *Zendia) AddMonitoringWithPersistence(collection *mongo.Collection) *Metrics {
	config := DefaultMetricsConfig
	
	// Tenta habilitar persistência, mas falha graciosamente
	metrics := NewMetricsWithConfig(config)
	
	// Testa conexão antes de habilitar persistência
	if collection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Testa se consegue acessar a collection
		if err := collection.Database().Client().Ping(ctx, nil); err != nil {
			fmt.Printf("Warning: MongoDB not available, disabling metrics persistence: %v\n", err)
			z.Use(Monitoring(metrics))
			return metrics
		}
		
		persister := NewMongoMetricsPersister(collection)
		
		// Tenta criar índices
		if err := persister.CreateIndexes(); err != nil {
			fmt.Printf("Warning: Failed to create metrics indexes: %v\n", err)
		}
		
		// Só habilita persistência se tudo funcionou
		config.EnablePersistence = true
		metrics.config = config
		metrics.SetPersister(persister)
		
		// Inicia rotina de persistência apenas se habilitada
		go metrics.startPersistenceRoutine()
		
		// Adiciona endpoints de histórico
		z.addMetricsHistoryEndpoints(metrics, persister)
		
		fmt.Println("✅ Metrics persistence enabled with MongoDB")
	} else {
		fmt.Println("⚠️  Metrics persistence disabled - no collection provided")
	}
	
	z.Use(Monitoring(metrics))
	return metrics
}

// addMetricsHistoryEndpoints adiciona endpoints para consultar histórico
func (z *Zendia) addMetricsHistoryEndpoints(metrics *Metrics, persister *MongoMetricsPersister) {
	// Endpoint para histórico de métricas
	z.GET("/public/metrics/history", Handle(func(c *Context[any]) error {
		// Parse query parameters
		fromStr := c.Query("from")
		toStr := c.Query("to")
		tenantID := c.Query("tenant_id")
		
		// Default: últimas 24 horas
		to := time.Now()
		from := to.Add(-24 * time.Hour)
		
		if fromStr != "" {
			if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
				from = parsed
			}
		}
		if toStr != "" {
			if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
				to = parsed
			}
		}
		
		history, err := persister.GetHistory(tenantID, from, to)
		if err != nil {
			return err
		}
		
		c.Success("Histórico de métricas", map[string]interface{}{
			"from":    from.Format(time.RFC3339),
			"to":      to.Format(time.RFC3339),
			"count":   len(history),
			"metrics": history,
		})
		return nil
	}))
	
	// Endpoint para estatísticas agregadas
	z.GET("/public/metrics/stats", Handle(func(c *Context[any]) error {
		fromStr := c.Query("from")
		toStr := c.Query("to")
		interval := c.DefaultQuery("interval", "hour") // hour, day, month
		tenantID := c.Query("tenant_id")
		
		// Default: últimas 24 horas
		to := time.Now()
		from := to.Add(-24 * time.Hour)
		
		if fromStr != "" {
			if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
				from = parsed
			}
		}
		if toStr != "" {
			if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
				to = parsed
			}
		}
		
		stats, err := persister.GetAggregatedStats(tenantID, from, to, interval)
		if err != nil {
			return err
		}
		
		c.Success("Estatísticas agregadas", map[string]interface{}{
			"from":     from.Format(time.RFC3339),
			"to":       to.Format(time.RFC3339),
			"interval": interval,
			"stats":    stats,
		})
		return nil
	}))
	
	// Endpoint para limpeza de dados antigos
	z.DELETE("/public/metrics/cleanup", Handle(func(c *Context[any]) error {
		daysStr := c.DefaultQuery("days", "30")
		days := 30
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
		
		if err := persister.Cleanup(days); err != nil {
			return err
		}
		
		c.Success("Limpeza realizada", map[string]interface{}{
			"removed_older_than_days": days,
		})
		return nil
	}))
}