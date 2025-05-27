package configs

import (
	"log"
	"os"
	"sync"
)

// AppConfig holds the application configuration.
// It's populated once by LoadConfig.
var AppConfig Configuration
var once sync.Once

// Configuration defines the structure for application settings.
type Configuration struct {
	JWTSecret       string
	ServerPort      string
	FrontendBaseURL string
}

const (
	defaultJWTSecret       = "mobile"                // Default JWT secret, used if env var is not set.
	envJWTSecretKey        = "JWT_SECRET_KEY"        // Environment variable name for the JWT secret.
	defaultServerPort      = "8081"                  // Default server port.
	envServerPortKey       = "SERVER_PORT"           // Environment variable name for the server port.
	defaultFrontendBaseURL = "http://localhost:3000" // 默认前端基础URL
	envFrontendBaseURLKey  = "FRONTEND_BASE_URL"     // 前端基础URL环境变量名
)

// LoadConfig loads configuration from environment variables or defaults.
// It should be called once at application startup.
func LoadConfig() {
	once.Do(func() {
		jwtSecret := os.Getenv(envJWTSecretKey)
		if jwtSecret == "" {
			jwtSecret = defaultJWTSecret
			log.Printf("警告: %s 环境变量未设置。正在使用默认的JWT密钥。请在生产环境中设置此变量以保证安全。", envJWTSecretKey)
		}

		serverPort := os.Getenv(envServerPortKey)
		if serverPort == "" {
			serverPort = defaultServerPort
			log.Printf("信息: %s 环境变量未设置。正在使用默认端口 %s。", envServerPortKey, defaultServerPort)
		}

		frontendBaseURL := os.Getenv(envFrontendBaseURLKey)
		if frontendBaseURL == "" {
			frontendBaseURL = defaultFrontendBaseURL
			log.Printf("信息: %s 环境变量未设置。正在使用默认前端URL %s。这在生产环境中可能不正确。", envFrontendBaseURLKey, defaultFrontendBaseURL)
		}

		AppConfig = Configuration{
			JWTSecret:       jwtSecret,
			ServerPort:      serverPort,
			FrontendBaseURL: frontendBaseURL,
		}

		log.Println("应用配置已加载。")
	})
}
