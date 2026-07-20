package http

import "github.com/gin-gonic/gin"

func Response(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

func ResponseError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
