package zendia

import (
	"fmt"
	"net/http"
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
func CORS(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
