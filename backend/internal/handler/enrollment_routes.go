package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/middleware"
)

// RegisterEnrollmentRoutes registers all enrollment-related routes.
// All endpoints require JWT authentication.
func RegisterEnrollmentRoutes(v1 *gin.RouterGroup, h *EnrollmentHandler, jwtSecret string) {
	enrollments := v1.Group("/enrollments", middleware.JWTAuth(jwtSecret))
	{
		enrollments.POST("", h.Create)
		enrollments.POST("/:id/cancel", h.Cancel)
		enrollments.GET("", h.List)
		enrollments.GET("/:id/status", h.GetStatus)
	}
}
