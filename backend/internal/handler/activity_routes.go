package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/uaad/backend/internal/middleware"
)

// RegisterActivityRoutes registers all activity-related routes.
// Public endpoints do not require authentication.
// B-end endpoints require JWT + MERCHANT role.
func RegisterActivityRoutes(v1 *gin.RouterGroup, h *ActivityHandler, jwtSecret string) {
	activities := v1.Group("/activities")
	{
		// Public endpoints (no auth)
		activities.GET("", h.List)
		// IMPORTANT: /merchant must be registered before /:id to avoid Gin matching "merchant" as an :id
		activities.GET("/merchant", middleware.JWTAuth(jwtSecret), middleware.RequireRole("MERCHANT"), h.MerchantList)
		activities.GET("/:id", h.Detail)
		activities.GET("/:id/stock", h.Stock)

		// B-end endpoints (JWT + MERCHANT)
		auth := activities.Group("", middleware.JWTAuth(jwtSecret), middleware.RequireRole("MERCHANT"))
		{
			auth.POST("", h.Create)
			auth.PUT("/:id", h.Update)
			auth.PUT("/:id/preheat", h.Preheat)
			auth.PUT("/:id/publish", h.Publish)
		}
	}
}
