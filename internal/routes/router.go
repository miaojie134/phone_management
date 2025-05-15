package routes

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes 初始化所有路由
func SetupRoutes(router *gin.Engine) {
	api := router.Group("/api")
	SetupAuthRoutes(api) // 注册认证路由
	// 未来可以在这里注册其他模块的路由，例如：
	// SetupEmployeeRoutes(api)
	// SetupMobileNumberRoutes(api)
}
