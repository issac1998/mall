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

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestUserRepository_Create(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		Phone:        "13800138000",
		PasswordHash: "hashedpassword",
		Salt:         "salt123",
		Status:       1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(ctx, user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := int64(1)
	expectedUser := &model.User{
		ID:           uint64(userID),
		Username:     "testuser",
		Phone:        "13800138000",
		PasswordHash: "hashedpassword",
		Salt:         "salt123",
		Status:       1,
	}

	rows := sqlmock.NewRows([]string{"id", "username", "phone", "password_hash", "salt", "status"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Phone, expectedUser.PasswordHash, expectedUser.Salt, expectedUser.Status)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE id = \\? ORDER BY `users`.`id` LIMIT \\?").
		WithArgs(userID, 1).
		WillReturnRows(rows)

	user, err := repo.GetByID(ctx, userID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}
	if user == nil {
		t.Error("Expected user, got nil")
		return
	}

	if user.ID != expectedUser.ID || user.Username != expectedUser.Username {
		t.Errorf("Expected user %+v, got %+v", expectedUser, user)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	username := "testuser"
	expectedUser := &model.User{
		ID:           1,
		Username:     username,
		Phone:        "13800138000",
		PasswordHash: "hashedpassword",
		Salt:         "salt123",
		Status:       1,
	}

	rows := sqlmock.NewRows([]string{"id", "username", "phone", "password_hash", "salt", "status"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Phone, expectedUser.PasswordHash, expectedUser.Salt, expectedUser.Status)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE username = \\? ORDER BY `users`.`id` LIMIT \\?").
		WithArgs(username, 1).
		WillReturnRows(rows)

	user, err := repo.GetByUsername(ctx, username)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}
	if user == nil {
		t.Error("Expected user, got nil")
		return
	}

	if user.Username != expectedUser.Username {
		t.Errorf("Expected username %s, got %s", expectedUser.Username, user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_GetByPhone(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138000"
	expectedUser := &model.User{
		ID:           1,
		Username:     "testuser",
		Phone:        phone,
		PasswordHash: "hashedpassword",
		Salt:         "salt123",
		Status:       1,
	}

	rows := sqlmock.NewRows([]string{"id", "username", "phone", "password_hash", "salt", "status"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Phone, expectedUser.PasswordHash, expectedUser.Salt, expectedUser.Status)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE phone = \\? ORDER BY `users`.`id` LIMIT \\?").
		WithArgs(phone, 1).
		WillReturnRows(rows)

	user, err := repo.GetByPhone(ctx, phone)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}
	if user == nil {
		t.Error("Expected user, got nil")
		return
	}

	if user.Phone != expectedUser.Phone {
		t.Errorf("Expected phone %s, got %s", expectedUser.Phone, user.Phone)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		ID:           1,
		Username:     "updateduser",
		Phone:        "13800138001",
		PasswordHash: "newhashedpassword",
		Salt:         "newsalt123",
		Status:       1,
		UpdatedAt:    time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `users` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Update(ctx, user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := int64(1)
	ip := "192.168.1.1"

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `users` SET `last_login_at`=\\?,`last_login_ip`=\\?,`updated_at`=\\? WHERE id = \\?").
		WithArgs(sqlmock.AnyArg(), ip, sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateLastLogin(ctx, userID, ip)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	username := "testuser"

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `users` WHERE username = \\?").
		WithArgs(username).
		WillReturnRows(rows)

	exists, err := repo.ExistsByUsername(ctx, username)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !exists {
		t.Errorf("Expected user to exist")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_ExistsByPhone(t *testing.T) {
	db, mock := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := NewUserRepository(db)
	ctx := context.Background()

	phone := "13800138000"

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `users` WHERE phone = \\?").
		WithArgs(phone).
		WillReturnRows(rows)

	exists, err := repo.ExistsByPhone(ctx, phone)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !exists {
		t.Errorf("Expected phone to exist")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUserRepository_Interface(t *testing.T) {
	db, _ := setupMockDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var _ UserRepository = NewUserRepository(db)
}