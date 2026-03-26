package db

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"mypan-backend/models"
)

var DB *gorm.DB

func InitDB() {
	var err error
	
	// 拦截未初始化就访问物理文件导致的“找不到路径”错误
	if err = os.MkdirAll("../data", 0755); err != nil {
		log.Fatalf("无法创建存放容器用的父数据夹: %v", err)
	}

	// 使用项目根目录下的 data/mypan.db
	DB, err = gorm.Open(sqlite.Open("../data/mypan.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	// 自动迁移 Scheme
	err = DB.AutoMigrate(&models.User{}, &models.Volume{}, &models.FileMeta{})
	if err != nil {
		log.Fatalf("执行数据库迁移失败: %v", err)
	}

	log.Println("数据库加载与迁移并完成")
}
