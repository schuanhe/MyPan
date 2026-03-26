package controllers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"mypan-backend/db"
	"mypan-backend/models"
	"mypan-backend/utils"
)

type VolumeReq struct {
	Name   string `json:"name" binding:"required"`
	Remark string `json:"remark"`
}

// CreateVolume 用户建立一个新的隔离性逻辑卷
func CreateVolume(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req VolumeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提交字段异常或缺失名称"})
		return
	}

	folderName := "vol_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	volDir, _ := utils.GetVolumeRealDir(folderName)

	if err := os.MkdirAll(volDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统层级建立物理文件夹失败"})
		return
	}

	// 生成唯一公开访问 key（即使是私有卷也预生成，方便后续开放）
	accessKey := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	vol := models.Volume{
		Name:         req.Name,
		FolderName:   folderName,
		Remark:       req.Remark,
		OwnerID:      userID,
		AccessMode:   models.VolumeAccessPrivate,
		AccessURLKey: accessKey,
	}

	if err := db.DB.Create(&vol).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库保存归档失败"})
		return
	}

	c.JSON(http.StatusOK, serializeVolume(vol))
}

// GetVolumes 返回我的卷资源列表
func GetVolumes(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	var vols []models.Volume
	if role == "admin" {
		db.DB.Find(&vols)
	} else {
		db.DB.Where("owner_id = ?", userID).Find(&vols)
	}

	result := make([]map[string]interface{}, len(vols))
	for i, v := range vols {
		result[i] = serializeVolume(v)
	}
	c.JSON(http.StatusOK, result)
}

// DeleteVolume 移除该卷及硬盘数据
func DeleteVolume(c *gin.Context) {
	id := c.Param("id")
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	var vol models.Volume
	if err := db.DB.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "此卷记录不复存在"})
		return
	}

	// 权限自证
	if role != "admin" && vol.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作他人的储存卷内层设计"})
		return
	}

	volDir, _ := utils.GetVolumeRealDir(vol.FolderName)
	os.RemoveAll(volDir) // 擦除本地

	db.DB.Delete(&vol)
	db.DB.Where("volume_id = ?", vol.ID).Delete(&models.FileMeta{}) // 清理文件元数据挂钩

	c.JSON(http.StatusOK, gin.H{"message": "指定存储卷已被焚毁"})
}

// VolumeAccessReq 设置开放卷的请求体
type VolumeAccessReq struct {
	AccessMode     string `json:"accessMode" binding:"required"` // private / public / login / password
	AccessPassword string `json:"accessPassword"`                // 仅 password 模式需要
	AccessURLKey   string `json:"accessURLKey"`                  // 自定义访问短码
}

// UpdateVolumeAccess 修改卷的对外访问权限
func UpdateVolumeAccess(c *gin.Context) {
	id := c.Param("id")
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	var req VolumeAccessReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var vol models.Volume
	if err := db.DB.First(&vol, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "卷不存在"})
		return
	}

	if role != "admin" && vol.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此卷"})
		return
	}

	mode := models.VolumeAccess(req.AccessMode)
	switch mode {
	case models.VolumeAccessPrivate, models.VolumeAccessPublic, models.VolumeAccessLogin, models.VolumeAccessPassword:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问模式"})
		return
	}

	vol.AccessMode = mode
	if mode == models.VolumeAccessPassword {
		if req.AccessPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密码模式必须设置密码"})
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.AccessPassword), bcrypt.DefaultCost)
		vol.AccessPassword = string(hash)
	} else {
		vol.AccessPassword = ""
	}

	// 处理自定义 Key
	if req.AccessURLKey != "" && req.AccessURLKey != vol.AccessURLKey {
		// 检查唯一性
		var count int64
		db.DB.Model(&models.Volume{}).Where("access_url_key = ? AND id <> ?", req.AccessURLKey, vol.ID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该访问短码已被其他储存卷占用"})
			return
		}
		// 也要检查 FileMeta 表是否占用了这个 Key
		db.DB.Model(&models.FileMeta{}).Where("access_url_key = ?", req.AccessURLKey).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该短码已被某个文件或文件夹分享占用"})
			return
		}
		vol.AccessURLKey = req.AccessURLKey
	}

	db.DB.Save(&vol)

	c.JSON(http.StatusOK, gin.H{
		"message":    "访问权限已更新",
		"accessMode": vol.AccessMode,
		"accessURL":  "/s/" + vol.AccessURLKey,
	})
}

// serializeVolume 序列化时使用小写 JSON key，确保前端一致
func serializeVolume(v models.Volume) map[string]interface{} {
	return map[string]interface{}{
		"id":           v.ID,
		"name":         v.Name,
		"remark":       v.Remark,
		"ownerID":      v.OwnerID,
		"accessMode":   string(v.AccessMode),
		"accessURLKey": v.AccessURLKey,
	}
}
