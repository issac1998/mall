package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"seckill/internal/model"
)

// UserRepository user repository interface
type UserRepository interface {
	// Create user
	Create(ctx context.Context, user *model.User) error

	// Get user by ID
	GetByID(ctx context.Context, id int64) (*model.User, error)

	// Get user by username
	GetByUsername(ctx context.Context, username string) (*model.User, error)

	// Get user by phone
	GetByPhone(ctx context.Context, phone string) (*model.User, error)

	// Get user by email
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// Update user info
	Update(ctx context.Context, user *model.User) error

	// Update last login info
	UpdateLastLogin(ctx context.Context, userID int64, ip string) error

	// Check if username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// Check if phone exists
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
}

// userRepository user repository implementation
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create creates a user
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID gets a user by ID
func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername gets a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByPhone gets a user by phone
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail gets a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update updates user info
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLastLogin updates last login info
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID int64, ip string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"last_login_ip": ip,
		}).Error
}

// ExistsByUsername checks if username exists
func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("username = ?", username).
		Count(&count).Error
	return count > 0, err
}

// ExistsByPhone checks if phone exists
func (r *userRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("phone = ?", phone).
		Count(&count).Error
	return count > 0, err
}

