package models

import "gorm.io/gorm"

// VolumeAccess 定义卷的对外访问模式
type VolumeAccess string

const (
	VolumeAccessPrivate  VolumeAccess = "private"  // 私有（默认），只有登录的 owner 可访问
	VolumeAccessPublic   VolumeAccess = "public"   // 完全开放，无需任何凭证
	VolumeAccessLogin    VolumeAccess = "login"    // 登录开放：携带有效 cookie/token 即可
	VolumeAccessPassword VolumeAccess = "password" // 密码访问：提交密码后获得 cookie 通行
)

// Volume 储存卷模型
type Volume struct {
	gorm.Model
	Name           string       `gorm:"not null"`
	FolderName     string       `gorm:"uniqueIndex;not null"` // 映射到系统 data 目录中的实体文件夹名
	Remark         string
	OwnerID        uint         `gorm:"index"`
	// 开放卷相关
	AccessMode     VolumeAccess `gorm:"default:'private'"`
	AccessURLKey   *string      `gorm:"uniqueIndex"` // 公开访问的短 URL key，如 /s/abc123
	AccessPassword string       // bcrypt hash，仅 password 模式使用
}
