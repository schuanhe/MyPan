package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"mypan-backend/db"
	"mypan-backend/models"
	"mypan-backend/utils"
)

type GenShareReq struct {
	VolumeID     uint   `json:"volumeId" binding:"required"`
	Path         string `json:"path" binding:"required"`
	Password     string `json:"password"`     // 如果为空，表示公共开放
	Days         int    `json:"days"`         // 有效天数，0表示永久
	AccessURLKey string `json:"accessURLKey"` // 可选，自定义短码
	AccessMode   string `json:"accessMode"`   // private / public / password / login
}

// GenerateShare 修改文件的权限属性以及生成提取秘密
func GenerateShare(c *gin.Context) {
	var req GenShareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提交字段未满足验证规范"})
		return
	}

	realPath, err := getValidatedRealDir(c, req.VolumeID, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	info, err := os.Stat(realPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目标物理文件已丢失"})
		return
	}

	var meta models.FileMeta
	res := db.DB.Where("volume_id = ? AND file_path = ?", req.VolumeID, req.Path).First(&meta)

	meta.VolumeID = req.VolumeID
	meta.FilePath = req.Path
	if info.IsDir() {
		meta.Type = "directory"
	} else {
		meta.Type = "file"
	}
	meta.Size = info.Size()

	if req.AccessMode == "password" && req.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		meta.PasswordHash = string(hash)
		meta.Permission = models.PermPassword
	} else if req.AccessMode == "login" {
		meta.Permission = models.PermLogin
		meta.PasswordHash = ""
	} else {
		meta.PasswordHash = ""
		meta.Permission = models.PermPublic
	}

	// 生成 AccessURLKey
	if req.Days == 0 && req.AccessURLKey != "" {
		// 检查自定义 Key 是否被占用 (排除当前 ID)
		var count int64
		db.DB.Model(&models.FileMeta{}).Where("access_url_key = ? AND id != ?", req.AccessURLKey, meta.ID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该自定义访问短码已被他人占用"})
			return
		}
		// 也要检查 Volume 表是否占用了这个 Key
		db.DB.Model(&models.Volume{}).Where("access_url_key = ?", req.AccessURLKey).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该短码已被某个储存卷占用"})
			return
		}
		meta.AccessURLKey = &req.AccessURLKey
	} else if meta.AccessURLKey == nil || *meta.AccessURLKey == "" {
		// 临时分享或未提供自定义 Key 时，使用随机 Key
		rk := utils.RandomString(8)
		meta.AccessURLKey = &rk
	}

	// 处理过期时间
	if req.Days > 0 {
		exp := time.Now().AddDate(0, 0, req.Days)
		meta.ExpiresAt = &exp
	} else {
		// 永久分享需清除过期时间
		meta.ExpiresAt = nil
	}

	if res.Error != nil {
		db.DB.Create(&meta)
	} else {
		db.DB.Save(&meta)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "分享权益挂载成功",
		"fileId":  meta.ID,
		"key":     meta.AccessURLKey,
	})
}

type ShareItem struct {
	ID           uint                  `json:"id"`
	Type         string                `json:"type"` // "volume", "file", "directory"
	Name         string                `json:"name"`
	Path         string                `json:"path"`
	AccessMode   string                `json:"accessMode"`
	AccessURLKey string                `json:"accessURLKey"`
	ExpiresAt    *time.Time            `json:"expiresAt"`
	VolumeID     uint                  `json:"volumeId"`
	VolumeName   string                `json:"volumeName"`
}

// ListShares 获取当前用户所有的分享项（卷与文件）
func ListShares(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	var list []ShareItem

	// 1. 获取卷分享
	var volumes []models.Volume
	query := db.DB.Where("access_mode != ?", models.VolumeAccessPrivate)
	if role != "admin" {
		query = query.Where("owner_id = ?", userID)
	}
	query.Find(&volumes)

	// 后置查询：将 VolumeID 映射到 Name，方便后续文件分享引用
	volMap := make(map[uint]string)
	for _, v := range volumes {
		volMap[v.ID] = v.Name
		list = append(list, ShareItem{
			ID:           v.ID,
			Type:         "volume",
			Name:         v.Name,
			Path:         "/",
			AccessMode:   string(v.AccessMode),
			AccessURLKey: utils.PtrToString(v.AccessURLKey),
			ExpiresAt:    nil,
			VolumeID:     v.ID,
			VolumeName:   v.Name,
		})
	}

	// 2. 获取文件/文件夹分享
	var files []models.FileMeta
	var myVolIDs []uint
	db.DB.Model(&models.Volume{}).Where("owner_id = ?", userID).Pluck("id", &myVolIDs)

	fQuery := db.DB.Where("permission != ?", models.PermPrivate)
	if role != "admin" {
		fQuery = fQuery.Where("volume_id IN ?", myVolIDs)
	}
	fQuery.Find(&files)

	for _, f := range files {
		vName, ok := volMap[f.VolumeID]
		if !ok {
			var v models.Volume
			db.DB.Select("name").First(&v, f.VolumeID)
			vName = v.Name
			volMap[f.VolumeID] = vName
		}
		list = append(list, ShareItem{
			ID:           f.ID,
			Type:         f.Type,
			Name:         filepath.Base(f.FilePath),
			Path:         f.FilePath,
			AccessMode:   string(f.Permission),
			AccessURLKey: utils.PtrToString(f.AccessURLKey),
			ExpiresAt:    f.ExpiresAt,
			VolumeID:     f.VolumeID,
			VolumeName:   vName,
		})
	}

	c.JSON(http.StatusOK, list)
}

// RevokeShare 撤销分享
func RevokeShare(c *gin.Context) {
	shareType := c.Param("type")
	id := c.Param("id")
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	if shareType == "volume" {
		var vol models.Volume
		if err := db.DB.First(&vol, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "卷未找到"})
			return
		}
		if role != "admin" && vol.OwnerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此卷"})
			return
		}
		db.DB.Model(&vol).Updates(map[string]interface{}{
			"access_mode":    models.VolumeAccessPrivate,
			"access_url_key": nil,
		})
	} else {
		var meta models.FileMeta
		if err := db.DB.First(&meta, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享记录未找到"})
			return
		}
		// 验证权限
		var vol models.Volume
		db.DB.First(&vol, meta.VolumeID)
		if role != "admin" && vol.OwnerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此分享"})
			return
		}

		db.DB.Model(&meta).Updates(map[string]interface{}{
			"permission":     models.PermPrivate,
			"access_url_key": nil,
			"expires_at":     nil,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "分享已撤销"})
}

// UpdateShareReq 更新分享要求的请求体
type UpdateShareReq struct {
	Password     string `json:"password"`
	Days         int    `json:"days"`         // 0 表示不变更或永久
	AccessURLKey string `json:"accessURLKey"` // 自定义短码
	AccessMode   string `json:"accessMode"`   // public / password / login
}

// UpdateShare 修改已有文件/文件夹分享的配置
func UpdateShare(c *gin.Context) {
	id := c.Param("id")
	userID := c.MustGet("userID").(uint)

	var req UpdateShareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var meta models.FileMeta
	if err := db.DB.First(&meta, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享记录不存在"})
		return
	}

	// 权限检查：需检查 Volume 所有权
	var vol models.Volume
	if err := db.DB.First(&vol, meta.VolumeID).Error; err != nil || vol.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此分享项"})
		return
	}

	// 处理 Key 变更
	if req.AccessURLKey != "" && (meta.AccessURLKey == nil || req.AccessURLKey != *meta.AccessURLKey) {
		var count int64
		db.DB.Model(&models.FileMeta{}).Where("access_url_key = ? AND id != ?", req.AccessURLKey, meta.ID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该自定义短码已被他人占用"})
			return
		}
		db.DB.Model(&models.Volume{}).Where("access_url_key = ?", req.AccessURLKey).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该短码已被某个储存卷占用"})
			return
		}
		meta.AccessURLKey = &req.AccessURLKey
	}

	// 处理模式与密码
	if req.AccessMode == "password" && req.Password != "" {
		meta.Permission = models.PermPassword
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		meta.PasswordHash = string(hash)
	} else if req.AccessMode == "login" {
		meta.Permission = models.PermLogin
		meta.PasswordHash = ""
	} else if req.AccessMode == "public" {
		meta.Permission = models.PermPublic
		meta.PasswordHash = ""
	}

	// 处理过期
	if req.Days > 0 {
		exp := time.Now().AddDate(0, 0, req.Days)
		meta.ExpiresAt = &exp
	} else if req.Days == -1 {
        // 约定 -1 为设为永久
        meta.ExpiresAt = nil
    }

	db.DB.Save(&meta)
	c.JSON(http.StatusOK, gin.H{"message": "分享配置已更新", "key": meta.AccessURLKey})
}

// PublicDownload 面向无关游客的外部直接请求 API，通过 fileId 索引与密码验证下发流资源
func PublicDownload(c *gin.Context) {
	fileID := c.Query("fileId")
	password := c.Query("password")

	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无明确的文件标识符"})
		return
	}

	var meta models.FileMeta
	if err := db.DB.First(&meta, fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "索引已断开或根本不存在"})
		return
	}

	if meta.Permission == models.PermPrivate {
		c.JSON(http.StatusForbidden, gin.H{"error": "所有权已被收敛至私有访问，您无权请求"})
		return
	}

	if meta.Permission == models.PermPassword {
		if password == "" || bcrypt.CompareHashAndPassword([]byte(meta.PasswordHash), []byte(password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "提取码错误，访问遭遇壁垒拦截"})
			return
		}
	}

	var vol models.Volume
	if err := db.DB.First(&vol, meta.VolumeID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "底层绑定卷发生系统偏离"})
		return
	}

	realPath, err := utils.GetFileRealDir(vol.FolderName, meta.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "后端处理文件地址时异常"})
		return
	}

	info, err := os.Stat(realPath)
	if os.IsNotExist(err) || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "真正的源被移除或者其已被替换为目录"})
		return
	}

	// 对外输出以纯正附加形式投递为妥，避免某些浏览器的内联问题
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(realPath))
	c.File(realPath)
}
