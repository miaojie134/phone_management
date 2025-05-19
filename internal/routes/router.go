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

		// --- 员工路由 (先初始化，因为 MobileNumberService 可能依赖它) ---
		employeeRepo := repositories.NewGormEmployeeRepository(db)
		employeeService := services.NewEmployeeService(employeeRepo)
		employeeHandler := handlers.NewEmployeeHandler(employeeService)

		// --- 手机号码路由 ---
		// 初始化手机号码相关的 repository, service, handler
		mobileNumberRepo := repositories.NewGormMobileNumberRepository(db)
		// 将 employeeService 注入到 MobileNumberService
		mobileNumberService := services.NewMobileNumberService(mobileNumberRepo, employeeService)
		mobileNumberHandler := handlers.NewMobileNumberHandler(mobileNumberService)

		mobileNumbersGroup := apiV1.Group("/mobilenumbers")
		mobileNumbersGroup.Use(jwtAuthMiddleware) // 对整个 /mobilenumbers 路由组应用 JWT 中间件
		{
			// POST /api/v1/mobilenumbers/
			mobileNumbersGroup.POST("/", mobileNumberHandler.CreateMobileNumber)
			// GET /api/v1/mobilenumbers/
			mobileNumbersGroup.GET("/", mobileNumberHandler.GetMobileNumbers)
			// GET /api/v1/mobilenumbers/:phoneNumber
			mobileNumbersGroup.GET("/:phoneNumber", mobileNumberHandler.GetMobileNumberByID)
			mobileNumbersGroup.POST("/:phoneNumber/update", mobileNumberHandler.UpdateMobileNumber)
			mobileNumbersGroup.POST("/:phoneNumber/assign", mobileNumberHandler.AssignMobileNumber)
			mobileNumbersGroup.POST("/:phoneNumber/unassign", mobileNumberHandler.UnassignMobileNumber)
		}

		// --- 员工路由组定义放在后面，但初始化已提前 ---
		employeeRoutes := apiV1.Group("/employees")
		employeeRoutes.Use(jwtAuthMiddleware)
		{
			employeeRoutes.POST("/", employeeHandler.CreateEmployee)
			employeeRoutes.GET("/", employeeHandler.GetEmployees)
			employeeRoutes.GET("/:employeeId", employeeHandler.GetEmployeeByID)
			// POST /api/v1/employees/:employeeId/update
			employeeRoutes.POST("/:employeeId/update", employeeHandler.UpdateEmployee)
			// POST /api/v1/employees/import -批量导入员工路由
			employeeRoutes.POST("/import", employeeHandler.BatchImportEmployees)
		}

	}

	// Swagger 文档路由 (如果使用 swaggo)
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
