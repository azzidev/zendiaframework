package zendia

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger middleware para logging de requisições
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return "[ZENDIA] " + param.TimeStamp.Format(time.RFC3339) +
			" | " + param.Method + " " + param.Path +
			" | " + param.ClientIP +
			" | " + fmt.Sprintf("%d", param.StatusCode) + " " + param.Latency.String() + "\n"
	})
}

// CORS middleware para Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// Compression middleware para compressão gzip
func Compression() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldCompress(c.Request) {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipWriter{c.Writer, gz}
		c.Next()
	}
}

// gzipWriter implementa gin.ResponseWriter com compressão
type gzipWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func shouldCompress(req *http.Request) bool {
	return strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")
}

// RateLimiter middleware básico para rate limiting
func RateLimiter(requests int, window time.Duration) gin.HandlerFunc {
	clients := make(map[string][]time.Time)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		
		// Limpa requisições antigas
		if times, exists := clients[clientIP]; exists {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < window {
					validTimes = append(validTimes, t)
				}
			}
			clients[clientIP] = validTimes
		}
		
		// Verifica limite
		if len(clients[clientIP]) >= requests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Rate limit exceeded",
			})
			c.Abort()
			return
		}
		
		// Adiciona requisição atual
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}

// Auth middleware básico para autenticação por token
func Auth(tokenValidator func(string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization token required",
			})
			c.Abort()
			return
		}
		
		// Remove "Bearer " prefix se presente
		if strings.HasPrefix(token, "Bearer ") {
			token = token[7:]
		}
		
		if !tokenValidator(token) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid token",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}