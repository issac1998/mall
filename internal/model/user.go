package model

import (
	"time"
)

// User model
type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement;comment:用户ID" json:"id"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null;comment:用户名" json:"username"`
	Phone        string     `gorm:"type:varchar(20);uniqueIndex;not null;comment:手机号" json:"phone"`
	Email        *string    `gorm:"type:varchar(100);uniqueIndex;comment:邮箱" json:"email,omitempty"`
	PasswordHash string     `gorm:"type:varchar(255);not null;comment:密码哈希" json:"-"`
	Salt         string     `gorm:"type:varchar(32);not null;comment:密码盐" json:"-"`
	Nickname     *string    `gorm:"type:varchar(50);comment:昵称" json:"nickname,omitempty"`
	Avatar       *string    `gorm:"type:varchar(255);comment:头像URL" json:"avatar,omitempty"`
	Gender       int8       `gorm:"type:tinyint;default:0;comment:性别：0-未知，1-男，2-女" json:"gender"`
	Birthday     *time.Time `gorm:"type:date;comment:生日" json:"birthday,omitempty"`
	Level        int        `gorm:"type:int;default:1;comment:用户等级" json:"level"`
	Points       int        `gorm:"type:int;default:0;comment:积分" json:"points"`
	Balance      int64      `gorm:"type:bigint;default:0;comment:余额（分）" json:"balance"`
	Status       int8       `gorm:"type:tinyint;not null;default:1;index;comment:状态：1-正常，2-禁用，3-注销" json:"status"`
	LastLoginAt  *time.Time `gorm:"type:timestamp;comment:最后登录时间" json:"last_login_at,omitempty"`
	LastLoginIP  *string    `gorm:"type:varchar(45);comment:最后登录IP" json:"last_login_ip,omitempty"`
	CreatedAt    time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;index;comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName set name	
func (User) TableName() string {
	return "users"
}

// UserStatus user status const
const (
	UserStatusNormal   = 1 // 正常
	UserStatusDisabled = 2 // 禁用
	UserStatusDeleted  = 3 // 注销
)

// UserGender user gender const
const (
	GenderUnknown = 0 // 未知
	GenderMale    = 1 // 男
	GenderFemale  = 2 // 女
)

// IsActive check if user is active	
func (u *User) IsActive() bool {
	return u.Status == UserStatusNormal
}

// IsDisabled check if user is disabled	
func (u *User) IsDisabled() bool {
	return u.Status == UserStatusDisabled
}

// IsDeleted check if user is deleted	
func (u *User) IsDeleted() bool {
	return u.Status == UserStatusDeleted
}