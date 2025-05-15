package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/routes"
	"github.com/phone_management/pkg/db"
)

func main() {
	// 初始化数据库连接
	db.InitDB()        // 从 pkg/db 调用 InitDB
	defer db.CloseDB() // 确保在 main 函数退出时关闭数据库连接

	router := gin.Default()

	// 设置API路由
	routes.SetupRoutes(router)

	// TODO: 从配置中读取端口号
	port := "8080"
	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
