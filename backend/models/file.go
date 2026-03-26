package models

import (
	"time"

	"gorm.io/gorm"
)

type PermissionType string

const (
	PermPrivate  PermissionType = "private"
	PermPublic   PermissionType = "public"
	PermPassword PermissionType = "password" // 访问需要密码
	PermLogin    PermissionType = "login"    // 访问需要登录
)

// FileMeta 文件元数据模型
// 用户可以选择对特定的分享链接开启密码保护等权限
type FileMeta struct {
	gorm.Model
	VolumeID     uint           `gorm:"index;not null"`
	FilePath     string         `gorm:"index;not null"` // 相对路径
	Size         int64
	Type         string         // "file" or "directory"
	Permission   PermissionType `gorm:"default:'private'"`
	PasswordHash string         // 空表示无密码
	// 新增分享字段
	AccessURLKey *string        `gorm:"uniqueIndex"` // 分享短链 Key
	ExpiresAt    *time.Time     // 失效时间，nil 表示永久
}
