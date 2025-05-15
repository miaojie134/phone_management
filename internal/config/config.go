package config

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
	JWTSecret  string
	ServerPort string
}

const (
	defaultJWTSecret  = "mobile"         // Default JWT secret, used if env var is not set.
	envJWTSecretKey   = "JWT_SECRET_KEY" // Environment variable name for the JWT secret.
	defaultServerPort = "8080"           // Default server port.
	envServerPortKey  = "SERVER_PORT"    // Environment variable name for the server port.
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

		AppConfig = Configuration{
			JWTSecret:  jwtSecret,
			ServerPort: serverPort,
		}

		log.Println("应用配置已加载。")
	})
}
