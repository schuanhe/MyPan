package routes

import (
	"github.com/gin-gonic/gin"
	"mypan-backend/controllers"
	"mypan-backend/middlewares"
)

func SetupRoutes(r *gin.Engine) {
	// 公开 API
	apiPublic := r.Group("/api/auth")
	{
		apiPublic.GET("/status", controllers.Status)
		apiPublic.POST("/register", controllers.Register)
		apiPublic.POST("/login", controllers.Login)
		apiPublic.GET("/share/download", controllers.PublicDownload)
	}

	// 需要 JWT 的保护性接口组
	apiAuth := r.Group("/api/v1")
	apiAuth.Use(middlewares.AuthMiddleware())
	{
		// Volume endpoints
		apiAuth.GET("/volumes", controllers.GetVolumes)
		apiAuth.POST("/volumes", controllers.CreateVolume)
		apiAuth.DELETE("/volumes/:id", controllers.DeleteVolume)
		apiAuth.PUT("/volumes/:id/access", controllers.UpdateVolumeAccess)

		// File endpoints
		apiAuth.GET("/files/list", controllers.ListFiles)
		apiAuth.POST("/files/folder", controllers.CreateFolder)
		apiAuth.PUT("/files/rename", controllers.RenameFile)
		apiAuth.DELETE("/files/delete", controllers.DeleteFile)
		apiAuth.POST("/files/upload", controllers.UploadFile)
		apiAuth.GET("/files/download", controllers.DownloadFile)

		// Share Configuration endpoint
		apiAuth.POST("/share/generate", controllers.GenerateShare)
		apiAuth.GET("/shares", controllers.ListShares)
		apiAuth.DELETE("/shares/:type/:id", controllers.RevokeShare)
		apiAuth.PUT("/shares/file/:id", controllers.UpdateShare)
	}

	// 公开卷访问路由 (无需 JWT 中间件)
	r.GET("/s/:key", controllers.PublicVolumeIndex)
	r.POST("/s/:key", controllers.PublicVolumeIndex)
	r.GET("/s/:key/download", controllers.PublicVolumeDownload)

	// 公开文件/文件夹分享路由
	r.GET("/f/:key", controllers.PublicFileIndex)
	r.POST("/f/:key", controllers.PublicFileIndex)
	r.GET("/f/:key/download", controllers.PublicFileDownload)

	r.GET("/login", controllers.PublicLogin)
	r.POST("/login", controllers.PublicLogin)
}
