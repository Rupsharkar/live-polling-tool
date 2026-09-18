package routes

import (
	"github.com/gin-gonic/gin"

	"live-polling-tool/backend/handlers"
	"live-polling-tool/backend/middleware"
	"live-polling-tool/backend/services"
)

func Register(r *gin.Engine, h *handlers.Handler, auth *middleware.Auth, redis *services.RedisService) {
	api := r.Group("/api")
	api.POST("/auth/signup", h.Signup)
	api.POST("/auth/login", h.Login)
	api.POST("/polls", auth.Require(), h.CreatePoll)
	api.GET("/polls/:id", h.GetPoll)
	api.POST("/polls/:id/vote", h.Vote)
	api.POST("/polls/:id/close", auth.Require(), h.ClosePoll)
	r.GET("/ws/polls/:id", h.WebSocket)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
}
