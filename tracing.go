package zendia

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

// TraceContext chaves para contexto de tracing
const (
	TraceIDKey     = "trace_id"
	SpanIDKey      = "span_id"
	ParentSpanKey  = "parent_span_id"
)

// Span representa um span de tracing
type Span struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Operation  string            `json:"operation"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Duration   time.Duration     `json:"duration"`
	Tags       map[string]string `json:"tags"`
	Status     string            `json:"status"`
}

// Tracer interface para implementações de tracing
type Tracer interface {
	StartSpan(ctx context.Context, operation string) (*Span, context.Context)
	FinishSpan(span *Span)
	InjectHeaders(ctx context.Context, headers map[string]string)
	ExtractHeaders(headers map[string]string) context.Context
}

// SimpleTracer implementação simples de tracing
type SimpleTracer struct {
	spans []Span
}

// NewSimpleTracer cria um novo tracer simples
func NewSimpleTracer() *SimpleTracer {
	return &SimpleTracer{
		spans: make([]Span, 0),
	}
}

// StartSpan inicia um novo span
func (t *SimpleTracer) StartSpan(ctx context.Context, operation string) (*Span, context.Context) {
	traceID := getTraceID(ctx)
	if traceID == "" {
		traceID = generateID()
	}
	
	parentSpanID := getSpanID(ctx)
	spanID := generateID()
	
	span := &Span{
		TraceID:   traceID,
		SpanID:    spanID,
		ParentID:  parentSpanID,
		Operation: operation,
		StartTime: time.Now(),
		Tags:      make(map[string]string),
		Status:    "started",
	}
	
	newCtx := context.WithValue(ctx, TraceIDKey, traceID)
	newCtx = context.WithValue(newCtx, SpanIDKey, spanID)
	
	return span, newCtx
}

// FinishSpan finaliza um span
func (t *SimpleTracer) FinishSpan(span *Span) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = "finished"
	t.spans = append(t.spans, *span)
}

// InjectHeaders injeta headers de tracing
func (t *SimpleTracer) InjectHeaders(ctx context.Context, headers map[string]string) {
	if traceID := getTraceID(ctx); traceID != "" {
		headers["X-Trace-ID"] = traceID
	}
	if spanID := getSpanID(ctx); spanID != "" {
		headers["X-Span-ID"] = spanID
	}
}

// ExtractHeaders extrai headers de tracing
func (t *SimpleTracer) ExtractHeaders(headers map[string]string) context.Context {
	ctx := context.Background()
	
	if traceID, exists := headers["X-Trace-ID"]; exists {
		ctx = context.WithValue(ctx, TraceIDKey, traceID)
	}
	if spanID, exists := headers["X-Span-ID"]; exists {
		ctx = context.WithValue(ctx, ParentSpanKey, spanID)
	}
	
	return ctx
}

// GetSpans retorna todos os spans coletados
func (t *SimpleTracer) GetSpans() []Span {
	return t.spans
}

// Tracing middleware para tracing automático
func Tracing(tracer Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extrai contexto dos headers
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		
		ctx := tracer.ExtractHeaders(headers)
		operation := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		
		span, newCtx := tracer.StartSpan(ctx, operation)
		
		// Adiciona tags do span
		span.Tags["http.method"] = c.Request.Method
		span.Tags["http.url"] = c.Request.URL.String()
		span.Tags["http.user_agent"] = c.Request.UserAgent()
		span.Tags["client.ip"] = c.ClientIP()
		
		// Adiciona contexto ao gin.Context
		c.Set("trace_context", newCtx)
		c.Set("current_span", span)
		
		c.Next()
		
		// Finaliza span com informações da resposta
		span.Tags["http.status_code"] = fmt.Sprintf("%d", c.Writer.Status())
		if c.Writer.Status() >= 400 {
			span.Status = "error"
		}
		
		tracer.FinishSpan(span)
	}
}

// GetTraceContext retorna o contexto de tracing do gin.Context
func GetTraceContext(c *gin.Context) context.Context {
	if ctx, exists := c.Get("trace_context"); exists {
		return ctx.(context.Context)
	}
	return context.Background()
}

// GetCurrentSpan retorna o span atual do gin.Context
func GetCurrentSpan(c *gin.Context) *Span {
	if span, exists := c.Get("current_span"); exists {
		return span.(*Span)
	}
	return nil
}

// Funções auxiliares
func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func getSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	if parentSpanID, ok := ctx.Value(ParentSpanKey).(string); ok {
		return parentSpanID
	}
	return ""
}

func generateID() string {
	return fmt.Sprintf("%016x", rand.Int63())
}