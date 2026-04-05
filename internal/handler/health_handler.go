package handler

import (
	"gin-app/internal/middleware"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":     "ok",
		"request_id": middleware.GetRequestID(c),
	})
}
