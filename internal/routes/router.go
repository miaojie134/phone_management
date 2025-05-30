package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/auth"
	"github.com/phone_management/internal/handlers"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/internal/services"
	"gorm.io/gorm"

	"github.com/gin-contrib/cors"        // 导入 CORS 包
	_ "github.com/phone_management/docs" // docs is generated by Swag CLI, you have to import it.
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Phone Management API
// @version 1.0
// @description This is a sample server for a phone management system.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8081
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// SetupRouter 配置所有应用路由
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	// 允许所有来源，您可以根据需要进行更严格的配置
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                                                   // 或者使用 config.AllowOrigins = []string{"http://example.com", "http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}                              // 根据需要添加或移除方法
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"} // 根据需要添加自定义Header
	config.AllowCredentials = true                                                                                  // 如果需要携带凭证（例如 cookies）
	r.Use(cors.New(config))

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

		// --- 员工路由 (先初始化，因为 MobileNumberService 依赖它) ---
		employeeRepo := repositories.NewGormEmployeeRepository(db)

		// --- 手机号码路由 ---
		// 初始化手机号码相关的 repository, service, handler
		mobileNumberRepo := repositories.NewGormMobileNumberRepository(db)

		// 初始化employeeService，现在需要mobileNumberRepo依赖
		employeeService := services.NewEmployeeService(employeeRepo, mobileNumberRepo)
		employeeHandler := handlers.NewEmployeeHandler(employeeService)

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
			// GET /api/v1/mobilenumbers/risk-pending - 获取风险号码列表
			mobileNumbersGroup.GET("/risk-pending", mobileNumberHandler.GetRiskPendingNumbers)
			// GET /api/v1/mobilenumbers/:phoneNumber
			mobileNumbersGroup.GET("/:phoneNumber", mobileNumberHandler.GetMobileNumberByID)
			mobileNumbersGroup.POST("/:phoneNumber/update", mobileNumberHandler.UpdateMobileNumber)
			mobileNumbersGroup.POST("/:phoneNumber/assign", mobileNumberHandler.AssignMobileNumber)
			mobileNumbersGroup.POST("/:phoneNumber/unassign", mobileNumberHandler.UnassignMobileNumber)
			// POST /api/v1/mobilenumbers/:phoneNumber/handle-risk - 处理风险号码
			mobileNumbersGroup.POST("/:phoneNumber/handle-risk", mobileNumberHandler.HandleRiskNumber)
			// POST /api/v1/mobilenumbers/import 批量导入手机号码
			mobileNumbersGroup.POST("/import", mobileNumberHandler.BatchImportMobileNumbers)
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
			// POST /api/v1/employees/import 批量导入员工
			employeeRoutes.POST("/import", employeeHandler.BatchImportEmployees)
		}

		// --- 号码验证路由 ---
		verificationTokenRepo := repositories.NewGormVerificationTokenRepository(db)
		verificationBatchTaskRepo := repositories.NewGormVerificationBatchTaskRepository(db)
		userReportedIssueRepo := repositories.NewGormUserReportedIssueRepository(db)
		submissionLogRepo := repositories.NewGormVerificationSubmissionLogRepository(db)
		verificationService := services.NewVerificationService(employeeRepo, verificationTokenRepo, verificationBatchTaskRepo, mobileNumberRepo, userReportedIssueRepo, submissionLogRepo, db)
		verificationHandler := handlers.NewVerificationHandler(verificationService)

		// 公开的验证接口，不需要JWT认证
		apiV1.GET("/verification/info", verificationHandler.GetVerificationInfo)
		apiV1.POST("/verification/submit", verificationHandler.SubmitVerificationResult)

		verificationGroup := apiV1.Group("/verification")
		verificationGroup.Use(jwtAuthMiddleware) // 对 /verification 路由组应用 JWT 中间件
		{
			// POST /api/v1/verification/initiate
			verificationGroup.POST("/initiate", verificationHandler.InitiateVerification)
			// GET /api/v1/verification/batch/{batchId}/status
			verificationGroup.GET("/batch/:batchId/status", verificationHandler.GetVerificationBatchStatus)
			// GET /api/v1/verification/admin/phone-status - 基于手机号维度的确认状态
			verificationGroup.GET("/admin/phone-status", verificationHandler.GetPhoneVerificationStatus)
			// 其他 /verification 子路由可以在这里添加，例如 GET /info, POST /submit, GET /admin/status
		}

	}

	// Swagger 文档路由 (如果使用 swaggo)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	return r
}
