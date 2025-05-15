package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/configs"
	"github.com/phone_management/internal/routes"
	"github.com/phone_management/pkg/db"
)

func main() {
	// 1. 加载应用配置
	// 这应该在任何依赖配置的代码之前执行
	configs.LoadConfig()

	// 2. 初始化数据库连接
	db.InitDB()        // 从 pkg/db 调用 InitDB
	defer db.CloseDB() // 确保在 main 函数退出时关闭数据库连接

	// 3. 初始化 Gin 引擎
	router := gin.Default()

	// 4. 设置API路由
	routes.SetupRoutes(router)

	// 5. 从配置中获取端口号并启动服务器
	port := configs.AppConfig.ServerPort // 使用配置中的端口
	log.Printf("服务器正在监听端口 %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
