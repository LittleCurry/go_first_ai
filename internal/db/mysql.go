package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// InitMySQL 初始化MySQL连接
func InitMySQL(cfg *config.MySQLConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql failed: %w", err)
	}

	// 设置连接池
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping mysql failed: %w", err)
	}

	log.Println("✅ MySQL connected successfully")
	return nil
}

// CloseMySQL 关闭MySQL连接
func CloseMySQL() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
