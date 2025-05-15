package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/handlers"
)

// SetupAuthRoutes 设置认证相关路由
func SetupAuthRoutes(router *gin.RouterGroup) {
	apiV1 := router.Group("/v1") // 创建 /api/v1 路由组
	{
		authGroup := apiV1.Group("/auth") // 在 /api/v1 下创建 /auth 路由组
		{
			// POST /api/v1/auth/login
			authGroup.POST("/login", handlers.Login)

			// POST /api/v1/auth/logout (根据文档)
			// authGroup.POST("/logout", handlers.Logout) // TODO: 实现Logout Handler后取消注释
		}
	}
}
