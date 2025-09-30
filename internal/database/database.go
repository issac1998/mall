package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"seckill/internal/config"
	"seckill/pkg/log"
)

var (
	DB *gorm.DB
)

// Init initialize database connection
func Init(cfg *config.Config) error {
	dsn := buildDSN(cfg.Database)
	
	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.GetLogger(),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  getLogLevel(cfg.Log.Level),
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// set connection pool
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Info("Database connected successfully")
	return nil
}

// Close close database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB get database connection instance
func GetDB() *gorm.DB {
	return DB
}

// buildDSN build MySQL connection string	
func buildDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)
}

// getLogLevel convert log level string to gorm logger.LogLevel
func getLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug":
		return logger.Info
	case "info":
		return logger.Warn
	case "warn":
		return logger.Error
	case "error":
		return logger.Error
	default:
		return logger.Warn
	}
}

// Health check database health status
func Health() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

// Transaction execute transaction
func Transaction(fn func(tx *gorm.DB) error) error {
	return DB.Transaction(fn)
}

// WithContext return database connection with context
func WithContext(ctx context.Context) *gorm.DB {
	if DB == nil {
		return nil
	}
	return DB.WithContext(ctx)
}