package database

import (
	"fmt"

	"gorm.io/gorm"

	"seckill/internal/model"
	"seckill/pkg/log"
)

// AutoMigrate auto migrate database table schema
func AutoMigrate(db *gorm.DB) error {
	log.Info("Starting database migration...")

	
	models := []interface{}{
		&model.User{},
		&model.Goods{},
		&model.SeckillActivity{},
		&model.Order{},
		&model.OrderDetail{},
		&model.StockLog{},
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
		log.Infof("Migrated model: %T", model)
	}

	log.Info("Database migration completed successfully")
	return nil
}

// CreateIndexes create additional indexes
func CreateIndexes(db *gorm.DB) error {
	log.Info("Creating additional indexes...")

	indexes := []struct {
		table string
		name  string
		sql   string
	}{
		{
			table: "seckill_activities",
			name:  "idx_activities_status_time",
			sql:   "CREATE INDEX IF NOT EXISTS idx_activities_status_time ON seckill_activities (status, start_time, end_time)",
		},
		{
			table: "orders",
			name:  "idx_orders_user_status",
			sql:   "CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders (user_id, status, created_at)",
		},
		{
			table: "stock_logs",
			name:  "idx_stock_logs_activity_time",
			sql:   "CREATE INDEX IF NOT EXISTS idx_stock_logs_activity_time ON stock_logs (activity_id, created_at)",
		},
	}

	for _, idx := range indexes {
		if err := db.Exec(idx.sql).Error; err != nil {
			log.Warnf("Failed to create index %s on table %s: %v", idx.name, idx.table, err)
		} else {
			log.Infof("Created index: %s on table %s", idx.name, idx.table)
		}
	}

	log.Info("Index creation completed")
	return nil
}

// DropTables drop all tables 
func DropTables(db *gorm.DB) error {
	log.Warn("Dropping all tables...")

	tables := []string{
		"stock_logs",
		"order_details",
		"orders",
		"seckill_activities",
		"goods",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Error; err != nil {
			log.Warnf("Failed to drop table %s: %v", table, err)
		} else {
			log.Infof("Dropped table: %s", table)
		}
	}

	log.Warn("All tables dropped")
	return nil
}

// CheckTables check if tables exist
func CheckTables(db *gorm.DB) error {
	log.Info("Checking database tables...")

	tables := []string{
		"users",
		"goods", 
		"seckill_activities",
		"orders",
		"order_details",
		"stock_logs",
	}

	for _, table := range tables {
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&count).Error
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		
		if count > 0 {
			log.Infof("Table exists: %s", table)
		} else {
			log.Warnf("Table not found: %s", table)
		}
	}

	log.Info("Table check completed")
	return nil
}