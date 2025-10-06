package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"seckill/internal/model"
)

// stringPtr returns a pointer to string
func stringPtr(s string) *string {
	return &s
}

func setupGoodsTestDB() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func TestGoodsRepository_Create(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goods := &model.Goods{
		Name:        "Test Goods",
		Description: stringPtr("Test Description"),
		Price:       10000,
		Stock:       100,
		Status:      model.GoodsStatusOnSale,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `goods`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), goods)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_GetByID(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goodsID := uint64(1)
	expectedGoods := &model.Goods{
		ID:     goodsID,
		Name:   "Test Goods",
		Price:  10000,
		Stock:  100,
		Status: model.GoodsStatusOnSale,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "price", "stock", "status"}).
		AddRow(expectedGoods.ID, expectedGoods.Name, expectedGoods.Price, expectedGoods.Stock, expectedGoods.Status)

	mock.ExpectQuery("SELECT \\* FROM `goods` WHERE id = \\? ORDER BY `goods`.`id` LIMIT \\?").
		WithArgs(goodsID, 1).
		WillReturnRows(rows)

	goods, err := repo.GetByID(context.Background(), goodsID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if goods == nil {
		t.Error("Expected goods, got nil")
		return
	}
	if goods.ID != expectedGoods.ID || goods.Name != expectedGoods.Name {
		t.Errorf("Expected goods %+v, got %+v", expectedGoods, goods)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goodsID := uint64(999)

	mock.ExpectQuery("SELECT \\* FROM `goods` WHERE id = \\? ORDER BY `goods`.`id` LIMIT \\?").
		WithArgs(goodsID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	goods, err := repo.GetByID(context.Background(), goodsID)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if goods != nil {
		t.Error("Expected nil goods, got non-nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_Update(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goods := &model.Goods{
		ID:          1,
		Name:        "Updated Goods",
		Description: stringPtr("Updated Description"),
		Price:       15000,
		Stock:       200,
		Status:      model.GoodsStatusOnSale,
		UpdatedAt:   time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `goods` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.Update(context.Background(), goods)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_UpdateStock(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goodsID := uint64(1)
	newStock := 150

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `goods` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.UpdateStock(context.Background(), goodsID, newStock)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_DecrStock(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goodsID := uint64(1)
	quantity := 5

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `goods` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.DecrStock(context.Background(), goodsID, quantity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_IncrStock(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goodsID := uint64(1)
	quantity := 5

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `goods` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.IncrStock(context.Background(), goodsID, quantity)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_List(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	page := 1
	pageSize := 10
	status := int8(model.GoodsStatusOnSale)

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `goods` WHERE status = \\?").
		WithArgs(status).
		WillReturnRows(countRows)

	// Mock data query
	rows := sqlmock.NewRows([]string{"id", "name", "price", "stock", "status"}).
		AddRow(1, "Goods 1", 10000, 100, status).
		AddRow(2, "Goods 2", 20000, 200, status)

	mock.ExpectQuery("SELECT \\* FROM `goods` WHERE status = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(status, pageSize).
		WillReturnRows(rows)

	goodsList, total, err := repo.List(context.Background(), page, pageSize, status)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(goodsList) != 2 {
		t.Errorf("Expected 2 goods, got %d", len(goodsList))
	}
	if goodsList[0].Name != "Goods 1" {
		t.Errorf("Expected first goods name 'Goods 1', got %s", goodsList[0].Name)
	}
	if goodsList[1].Name != "Goods 2" {
		t.Errorf("Expected second goods name 'Goods 2', got %s", goodsList[1].Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepository_List_EmptyResult(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	page := 1
	pageSize := 10
	status := int8(model.GoodsStatusOnSale)

	// Mock count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `goods` WHERE status = \\?").
		WithArgs(status).
		WillReturnRows(countRows)

	// Mock data query
	rows := sqlmock.NewRows([]string{"id", "name", "price", "stock", "status"})

	mock.ExpectQuery("SELECT \\* FROM `goods` WHERE status = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(status, pageSize).
		WillReturnRows(rows)

	goodsList, total, err := repo.List(context.Background(), page, pageSize, status)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
	if len(goodsList) != 0 {
		t.Errorf("Expected 0 goods, got %d", len(goodsList))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGoodsRepositoryInterface(t *testing.T) {
	db, _, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var _ GoodsRepository = NewGoodsRepository(db)
}

func TestGoodsRepository_DatabaseError(t *testing.T) {
	db, mock, err := setupGoodsTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewGoodsRepository(db)

	goods := &model.Goods{
		Name:        "Test Goods",
		Description: stringPtr("Test Description"),
		Price:       10000,
		Stock:       100,
		Status:      model.GoodsStatusOnSale,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `goods`").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err = repo.Create(context.Background(), goods)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}