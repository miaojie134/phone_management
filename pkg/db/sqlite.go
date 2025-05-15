package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var DB *sql.DB

const (
	dbDriver         = "sqlite3"
	defaultDbPathEnv = "SQLITE_DB_PATH"
	defaultDbFile    = "data/enterprise_mobile.db"
)

// InitDB 初始化数据库连接
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
	DB, err = sql.Open(dbDriver, dbPath)
	if err != nil {
		log.Fatalf("Failed to open database connection to %s: %v", dbPath, err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database %s: %v", dbPath, err)
	}

	log.Printf("Successfully connected to database: %s", dbPath)
	// 在这里可以考虑执行数据库迁移脚本
	// e.g., runMigrations(DB)
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		log.Println("Database connection closed.")
	}
}
