package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/pkg/db"
)

func main() {
	// 初始化数据库连接
	db.InitDB()        // 从 pkg/db 调用 InitDB
	defer db.CloseDB() // 确保在 main 函数退出时关闭数据库连接

	router := gin.Default()

	// 设置API路由
	// apiV1 := router.Group("/api/v1")
	// routes.SetupAuthRoutes(apiV1)       // 认证路由
	// routes.SetupEmployeeRoutes(apiV1)   // 员工路由
	// routes.SetupMobileNumberRoutes(apiV1) // 手机号路由
	// routes.SetupImportRoutes(apiV1)     // 导入路由

	log.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
