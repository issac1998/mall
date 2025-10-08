package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"seckill/internal/model"
)

func setupActivityMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create GORM database: %v", err)
	}

	return gormDB, mock
}

func TestActivityRepository_Create(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	activity := &model.SeckillActivity{
		Name:      "Test Activity",
		GoodsID:   1,
		Price:     10000,
		Stock:     100,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(24 * time.Hour),
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `seckill_activities`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(ctx, activity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_GetByID(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activityID := int64(1)
	expectedActivity := &model.SeckillActivity{
		ID:      uint64(activityID),
		Name:    "Test Activity",
		GoodsID: 1,
		Status:  1,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "goods_id", "status"}).
		AddRow(expectedActivity.ID, expectedActivity.Name, expectedActivity.GoodsID, expectedActivity.Status)

	mock.ExpectQuery("SELECT \\* FROM `seckill_activities` WHERE id = \\? ORDER BY `seckill_activities`.`id` LIMIT \\?").
		WithArgs(activityID, 1).
		WillReturnRows(rows)

	activity, err := repo.GetByID(context.Background(), activityID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if activity == nil {
		t.Error("Expected activity, got nil")
		return
	}
	if activity.ID != expectedActivity.ID || activity.Name != expectedActivity.Name {
		t.Errorf("Expected activity %+v, got %+v", expectedActivity, activity)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_GetByIDWithGoods(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activityID := int64(1)
	expectedActivity := &model.SeckillActivity{
		ID:      uint64(activityID),
		Name:    "Test Activity",
		GoodsID: 1,
		Status:  1,
	}

	// Mock activity query with preload - 注意使用正确的列名
	activityRows := sqlmock.NewRows([]string{"id", "name", "product_id", "seckill_price", "seckill_stock", "start_time", "end_time", "limit_per_user", "status", "created_at", "updated_at"}).
		AddRow(expectedActivity.ID, expectedActivity.Name, expectedActivity.GoodsID, 99.99, 100, time.Now(), time.Now(), 5, expectedActivity.Status, time.Now(), time.Now())

	mock.ExpectQuery("SELECT \\* FROM `seckill_activities` WHERE id = \\? ORDER BY `seckill_activities`.`id` LIMIT \\?").
		WithArgs(activityID, 1).
		WillReturnRows(activityRows)

	// Mock goods query for preload - 注意price是int64类型（分）
	goodsRows := sqlmock.NewRows([]string{"id", "name", "price", "stock", "description", "category", "brand", "images", "status", "sales", "created_at", "updated_at"}).
		AddRow(1, "Test Goods", 9999, 100, "Test Description", "Test Category", "Test Brand", "[]", 1, 0, time.Now(), time.Now())

	mock.ExpectQuery("SELECT \\* FROM `goods` WHERE `goods`.`id` = \\?").
		WithArgs(expectedActivity.GoodsID).
		WillReturnRows(goodsRows)

	activity, err := repo.GetByIDWithGoods(context.Background(), activityID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if activity == nil {
		t.Error("Expected activity, got nil")
		return
	}
	if activity.ID != expectedActivity.ID || activity.Name != expectedActivity.Name {
		t.Errorf("Expected activity %+v, got %+v", expectedActivity, activity)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_Update(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activity := &model.SeckillActivity{
		ID:        1,
		Name:      "Updated Activity",
		GoodsID:   1,
		Price:     15000,
		Stock:     200,
		Status:    1,
		UpdatedAt: time.Now(),
	}

	mock.ExpectBegin()
	// 简化SQL匹配，只匹配关键部分
	mock.ExpectExec("UPDATE `seckill_activities`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Update(context.Background(), activity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_UpdateStatus(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activityID := int64(1)
	newStatus := int8(2)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `seckill_activities` SET `status`=\\?,`updated_at`=\\? WHERE id = \\?").
		WithArgs(newStatus, sqlmock.AnyArg(), activityID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStatus(context.Background(), activityID, newStatus)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_ListActive(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	page := 1
	pageSize := 10

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `seckill_activities` WHERE status = \\? AND start_time <= \\? AND end_time >= \\?").
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(countRows)

	// Mock data query
	rows := sqlmock.NewRows([]string{"id", "name", "goods_id", "status"}).
		AddRow(1, "Activity 1", 1, 1).
		AddRow(2, "Activity 2", 2, 1)

	mock.ExpectQuery("SELECT \\* FROM `seckill_activities` WHERE status = \\? AND start_time <= \\? AND end_time >= \\? ORDER BY start_time ASC LIMIT \\?").
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg(), pageSize).
		WillReturnRows(rows)

	activities, total, err := repo.ListActive(context.Background(), page, pageSize)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(activities) != 2 {
		t.Errorf("Expected 2 activities, got %d", len(activities))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_DecrStock(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activityID := int64(1)
	quantity := 1

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `seckill_activities` SET `sold`=sold \\+ \\?,`stock`=stock - \\?,`updated_at`=\\? WHERE id = \\? AND stock >= \\?").
		WithArgs(quantity, quantity, sqlmock.AnyArg(), uint64(activityID), quantity).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.DecrStock(context.Background(), activityID, quantity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestActivityRepository_IncrStock(t *testing.T) {
	db, mock := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewActivityRepository(db)

	activityID := int64(1)
	quantity := 1

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `seckill_activities` SET `sold`=sold - \\?,`stock`=stock \\+ \\?,`updated_at`=\\? WHERE id = \\?").
		WithArgs(quantity, quantity, sqlmock.AnyArg(), uint64(activityID)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.IncrStock(context.Background(), activityID, quantity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// Test interface compliance
func TestActivityRepository_Interface(t *testing.T) {
	db, _ := setupActivityMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var _ ActivityRepository = NewActivityRepository(db)
}