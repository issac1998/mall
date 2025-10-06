package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"seckill/internal/model"
)

func setupStockLogTestDB() (*gorm.DB, sqlmock.Sqlmock, error) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	return gormDB, mock, nil
}

func TestStockLogRepository_Create(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	orderNo := "ORDER123456"
	requestID := "REQ123456"
	operator := "system"
	remark := "秒杀扣减库存"

	log := &model.StockLog{
		ActivityID:    1,
		GoodsID:       1,
		OperationType: int8(model.OperationTypeDeduct),
		Quantity:      -1,
		BeforeStock:   100,
		AfterStock:    99,
		RequestID:     &requestID,
		OrderNo:       &orderNo,
		Operator:      &operator,
		Remark:        &remark,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `stock_logs`").
		WithArgs(log.ActivityID, log.GoodsID, log.OperationType, log.Quantity, log.BeforeStock, log.AfterStock, log.RequestID, log.OrderNo, log.Operator, log.Remark).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), log)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStockLogRepository_GetByActivityID(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	activityID := uint64(1)
	limit := 10

	rows := sqlmock.NewRows([]string{"id", "activity_id", "goods_id", "operation_type", "quantity", "before_stock", "after_stock"}).
		AddRow(1, activityID, 1, model.OperationTypeDeduct, -1, 100, 99).
		AddRow(2, activityID, 1, model.OperationTypeRevert, 1, 99, 100)

	mock.ExpectQuery("SELECT \\* FROM `stock_logs` WHERE activity_id = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(activityID, limit).
		WillReturnRows(rows)

	logs, err := repo.GetByActivityID(context.Background(), activityID, limit)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, activityID, logs[0].ActivityID)
	assert.Equal(t, int8(model.OperationTypeDeduct), logs[0].OperationType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStockLogRepository_GetByOrderNo(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	orderNo := "ORDER123456"
	expectedLog := &model.StockLog{
		ID:            1,
		ActivityID:    1,
		GoodsID:       1,
		OperationType: model.OperationTypeDeduct,
		Quantity:      -1,
		BeforeStock:   100,
		AfterStock:    99,
		OrderNo:       &orderNo,
	}

	rows := sqlmock.NewRows([]string{"id", "activity_id", "goods_id", "operation_type", "quantity", "before_stock", "after_stock", "order_no"}).
		AddRow(expectedLog.ID, expectedLog.ActivityID, expectedLog.GoodsID, expectedLog.OperationType, expectedLog.Quantity, expectedLog.BeforeStock, expectedLog.AfterStock, expectedLog.OrderNo)

	mock.ExpectQuery("SELECT \\* FROM `stock_logs` WHERE order_no = \\? ORDER BY `stock_logs`.`id` LIMIT \\?").
		WithArgs(orderNo, 1).
		WillReturnRows(rows)

	log, err := repo.GetByOrderNo(context.Background(), orderNo)
	assert.NoError(t, err)
	assert.Equal(t, expectedLog.ID, log.ID)
	assert.Equal(t, expectedLog.ActivityID, log.ActivityID)
	assert.Equal(t, orderNo, *log.OrderNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStockLogRepository_GetByOrderNo_NotFound(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	orderNo := "NOTFOUND123"

	mock.ExpectQuery("SELECT \\* FROM `stock_logs` WHERE order_no = \\? ORDER BY `stock_logs`.`id` LIMIT \\?").
		WithArgs(orderNo, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	log, err := repo.GetByOrderNo(context.Background(), orderNo)
	assert.Error(t, err)
	assert.Nil(t, log)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStockLogRepository_List(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	page := 1
	pageSize := 10

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `stock_logs`").
		WillReturnRows(countRows)

	// Mock data query
	rows := sqlmock.NewRows([]string{"id", "activity_id", "goods_id", "operation_type", "quantity", "before_stock", "after_stock"}).
		AddRow(1, 1, 1, int8(model.OperationTypeDeduct), -1, 100, 99).
		AddRow(2, 1, 1, int8(model.OperationTypeRevert), 1, 99, 100)

	mock.ExpectQuery("SELECT \\* FROM `stock_logs` ORDER BY created_at DESC LIMIT \\?").
		WithArgs(pageSize).
		WillReturnRows(rows)

	logs, total, err := repo.List(context.Background(), page, pageSize)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
	assert.Equal(t, int8(model.OperationTypeDeduct), logs[0].OperationType)
	assert.Equal(t, int8(model.OperationTypeRevert), logs[1].OperationType)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStockLogRepository_List_EmptyResult(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	page := 1
	pageSize := 10

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `stock_logs`").
		WillReturnRows(countRows)

	// Mock empty data query
	rows := sqlmock.NewRows([]string{"id", "activity_id", "goods_id", "operation_type", "quantity", "before_stock", "after_stock"})
	mock.ExpectQuery("SELECT \\* FROM `stock_logs` ORDER BY created_at DESC LIMIT \\?").
		WithArgs(pageSize).
		WillReturnRows(rows)

	logs, total, err := repo.List(context.Background(), page, pageSize)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, logs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test interface compliance
func TestStockLogRepositoryInterface(t *testing.T) {
	db, _, err := setupStockLogTestDB()
	assert.NoError(t, err)

	var _ StockLogRepository = NewStockLogRepository(db)
}

// Test error handling
func TestStockLogRepository_DatabaseError(t *testing.T) {
	db, mock, err := setupStockLogTestDB()
	assert.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewStockLogRepository(db)

	log := &model.StockLog{
		ActivityID:    1,
		GoodsID:       1,
		OperationType: model.OperationTypeDeduct,
		Quantity:      -1,
		BeforeStock:   100,
		AfterStock:    99,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `stock_logs`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err = repo.Create(context.Background(), log)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test operation type validation
func TestStockLogOperationTypes(t *testing.T) {
	log := &model.StockLog{
		OperationType: model.OperationTypeDeduct,
	}
	assert.True(t, log.IsDeduct())
	assert.False(t, log.IsRevert())
	assert.False(t, log.IsSync())

	log.OperationType = model.OperationTypeRevert
	assert.False(t, log.IsDeduct())
	assert.True(t, log.IsRevert())
	assert.False(t, log.IsSync())

	log.OperationType = model.OperationTypeSync
	assert.False(t, log.IsDeduct())
	assert.False(t, log.IsRevert())
	assert.True(t, log.IsSync())
}