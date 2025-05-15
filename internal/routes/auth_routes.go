package routes

import (
	// 为示例代码添加 http 包导入

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/auth" // 导入JWT中间件包
	"github.com/phone_management/internal/handlers"
)

// SetupAuthRoutes 设置认证相关路由
func SetupAuthRoutes(router *gin.RouterGroup) {
	apiV1 := router.Group("/v1") // 创建 /api/v1 路由组
	{
		// 公共认证路由组 (例如登录)
		publicAuthGroup := apiV1.Group("/auth")
		{
			// POST /api/v1/auth/login
			publicAuthGroup.POST("/login", handlers.Login)
		}

		// 受保护的认证路由组 (例如登出)
		protectedAuthGroup := apiV1.Group("/auth")
		protectedAuthGroup.Use(auth.JWTMiddleware()) // 应用JWT中间件到这个组
		{
			// POST /api/v1/auth/logout
			protectedAuthGroup.POST("/logout", handlers.LogoutHandler)
		}

		// 示例：其他需要JWT认证的路由组
		// securedDataGroup := apiV1.Group("/data") // 假设是 /api/v1/data
		// securedDataGroup.Use(auth.JWTMiddleware()) // 应用JWT中间件
		// {
		// 	 securedDataGroup.GET("/me", func(c *gin.Context) {
		// 		 userID := c.MustGet("userID").(uint)
		// 		 username := c.MustGet("username").(string)
		// 		 role := c.MustGet("role").(string)
		// 		 jti := c.GetString("jti")
		// 		 c.JSON(http.StatusOK, gin.H{
		// 			 "message": "This is a protected route",
		// 			 "userID": userID,
		// 			 "username": username,
		// 			 "role": role,
		// 			 "jti": jti,
		// 		 })
		// 	 })
		// }
	}
}
