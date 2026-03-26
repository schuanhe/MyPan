package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"mypan-backend/db"
	"mypan-backend/routes"
)

func main() {
	// 1. 建立数据库连接并执行迁移
	db.InitDB()

	// 2. 建立 Gin
	r := gin.Default()

	// 解决携带 Token 头和 JSON 时的浏览器 CORS 拦截问题
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// 3. 注册所有路由
	routes.SetupRoutes(r)

	// 4. 监听
	log.Println("MyPan Backend 启动成功，监听 :8080 ...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Web 服务器启动失败: %v", err)
	}
}
