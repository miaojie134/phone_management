package db

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/phone_management/internal/models"
)

var gormDB *gorm.DB

const (
	defaultDbPathEnv = "SQLITE_DB_PATH"
	defaultDbFile    = "data/enterprise_mobile.db"
)

// InitDB 初始化 GORM 数据库连接
// 数据库文件路径通过环境变量 SQLITE_DB_PATH 获取，如果未设置，则使用默认值 "data/enterprise_mobile.db"
func InitDB() {
	dbPath := os.Getenv(defaultDbPathEnv)
	if dbPath == "" {
		dbPath = defaultDbFile
		log.Printf("Environment variable %s not set, using default database path: %s", defaultDbPathEnv, dbPath)
	} else {
		log.Printf("Using database path from environment variable %s: %s", defaultDbPathEnv, dbPath)
	}

	// 确保数据库文件所在的目录存在
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Database directory %s does not exist, creating it...", dbDir)
		if mkErr := os.MkdirAll(dbDir, 0755); mkErr != nil {
			log.Fatalf("Failed to create database directory %s: %v", dbDir, mkErr)
		}
	}

	var err error
	// 配置 GORM 日志级别
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  false,       // 禁用彩色打印
		},
	)

	gormDB, err = gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_timeout=20000&_synchronous=NORMAL&_cache_size=1000&_locking_mode=NORMAL&_temp_store=memory"), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Fatalf("Failed to connect to database %s: %v", dbPath, err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB from GORM: %v", err)
	}

	// 设置数据库连接池参数 (优化并发性能)
	sqlDB.SetMaxIdleConns(5)  // 减少空闲连接数
	sqlDB.SetMaxOpenConns(25) // 减少最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 设置SQLite特定的PRAGMA参数来优化并发性能
	if err := gormDB.Exec("PRAGMA busy_timeout = 30000").Error; err != nil {
		log.Printf("Warning: Failed to set busy_timeout: %v", err)
	}
	if err := gormDB.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		log.Printf("Warning: Failed to set journal_mode: %v", err)
	}
	if err := gormDB.Exec("PRAGMA synchronous = NORMAL").Error; err != nil {
		log.Printf("Warning: Failed to set synchronous: %v", err)
	}
	if err := gormDB.Exec("PRAGMA cache_size = 1000").Error; err != nil {
		log.Printf("Warning: Failed to set cache_size: %v", err)
	}
	if err := gormDB.Exec("PRAGMA temp_store = memory").Error; err != nil {
		log.Printf("Warning: Failed to set temp_store: %v", err)
	}

	log.Printf("Successfully connected to database using GORM: %s", dbPath)

	// 自动迁移数据库表结构
	err = gormDB.AutoMigrate(
		&models.User{},
		&models.Employee{},
		&models.MobileNumber{},
		&models.NumberUsageHistory{},
		&models.NumberApplicantHistory{},
		&models.VerificationToken{},
		&models.UserReportedIssue{},
		&models.VerificationBatchTask{},
		&models.VerificationSubmissionLog{},
	)
	if err != nil {
		log.Fatalf("Failed to auto migrate database tables: %v", err)
	}
	log.Println("Database tables migrated successfully.")
}

// GetDB 返回 GORM 数据库实例
func GetDB() *gorm.DB {
	if gormDB == nil {
		log.Fatal("Database not initialized. Call InitDB first.")
	}
	return gormDB
}

// CloseDB 关闭 GORM 数据库连接 (通常在应用退出时调用)
func CloseDB() {
	if gormDB != nil {
		sqlDB, err := gormDB.DB()
		if err != nil {
			log.Printf("Error getting underlying sql.DB for closing: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		log.Println("Database connection closed.")
	}
}
