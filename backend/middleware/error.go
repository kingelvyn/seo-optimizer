package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ErrorHandler middleware recovers from any panics and handles errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				log.Printf("Panic recovered: %v\nStack trace:\n%s", err, debug.Stack())

				// Return a 500 error to the client
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "An unexpected error occurred",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
} 