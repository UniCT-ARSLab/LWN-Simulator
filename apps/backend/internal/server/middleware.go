package server

import "github.com/gin-gonic/gin"

func registerMiddleware(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	})
}
