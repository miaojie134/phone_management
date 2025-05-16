package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/auth"
	"github.com/phone_management/internal/handlers"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/internal/services"
	"gorm.io/gorm"
)

// SetupRouter 配置所有应用路由
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// CORS 中间件 (如果需要)
	// r.Use(cors.Default())

	// JWT 中间件实例化
	jwtAuthMiddleware := auth.JWTMiddleware()

	// 创建 /api/v1 路由组
	apiV1 := r.Group("/api/v1")
	{
		// --- 认证路由 ---
		authGroup := apiV1.Group("/auth")
		{
			// POST /api/v1/auth/login - 公开路由，不需要JWT
			authGroup.POST("/login", handlers.Login)

			// POST /api/v1/auth/logout - 受保护路由，需要JWT
			// LogoutHandler 内部会处理JTI，所以应用JWT中间件
			authGroup.POST("/logout", jwtAuthMiddleware, handlers.LogoutHandler)
		}

		// --- 手机号码路由 ---
		// 初始化手机号码相关的 repository, service, handler
		mobileNumberRepo := repositories.NewGormMobileNumberRepository(db)
		mobileNumberService := services.NewMobileNumberService(mobileNumberRepo)
		mobileNumberHandler := handlers.NewMobileNumberHandler(mobileNumberService)

		mobileNumbersGroup := apiV1.Group("/mobilenumbers")
		mobileNumbersGroup.Use(jwtAuthMiddleware) // 对整个 /mobilenumbers 路由组应用 JWT 中间件
		{
			// POST /api/v1/mobilenumbers/
			mobileNumbersGroup.POST("/", mobileNumberHandler.CreateMobileNumber)
			mobileNumbersGroup.GET("/", mobileNumberHandler.GetMobileNumbers)
			// mobileNumbersGroup.GET("/:id", mobileNumberHandler.GetMobileNumberByID)
			// mobileNumbersGroup.POST("/:id/update", mobileNumberHandler.UpdateMobileNumber)
			// mobileNumbersGroup.POST("/:id/assign", mobileNumberHandler.AssignMobileNumber)
			// mobileNumbersGroup.POST("/:id/unassign", mobileNumberHandler.UnassignMobileNumber)
		}

		// --- 员工路由 (示例) ---
		// employeeRepo := repositories.NewGormEmployeeRepository(db)
		// employeeService := services.NewEmployeeService(employeeRepo)
		// employeeHandler := handlers.NewEmployeeHandler(employeeService)
		//
		// employeeRoutes := apiV1.Group("/employees")
		// employeeRoutes.Use(jwtAuthMiddleware)
		// {
		// // employeeRoutes.POST("/", employeeHandler.CreateEmployee)
		// // employeeRoutes.GET("/", employeeHandler.GetEmployees)
		// }

		// 示例：其他需要JWT认证的路由组 (如 auth_routes.go 中曾有的 /data/me 示例)
		/*
			securedDataGroup := apiV1.Group("/data")
			securedDataGroup.Use(jwtAuthMiddleware)
			{
				securedDataGroup.GET("/me", func(c *gin.Context) {
					userID := c.MustGet("userID").(uint)
					username := c.MustGet("username").(string)
					role := c.MustGet("role").(string)
					jti := c.GetString("jti")
					c.JSON(http.StatusOK, gin.H{
						"message":  "This is a protected route",
						"userID":   userID,
						"username": username,
						"role":     role,
						"jti":      jti,
					})
				})
			}
		*/
	}

	// Swagger 文档路由 (如果使用 swaggo)
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
