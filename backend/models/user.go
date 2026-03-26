package models

import "gorm.io/gorm"

// User 系统用户模型
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null;default:'user'"` // admin 设置权限, user 为普通用户
}
