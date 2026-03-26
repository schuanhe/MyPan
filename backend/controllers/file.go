package controllers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"mypan-backend/db"
	"mypan-backend/models"
	"mypan-backend/utils"
)

type PathReq struct {
	VolumeID uint   `form:"volumeId" binding:"required"`
	Path     string `form:"path"` // 选填，为空代表根目录
}

type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

func getValidatedRealDir(c *gin.Context, volID uint, relPath string) (string, error) {
	userID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)

	var vol models.Volume
	if err := db.DB.First(&vol, volID).Error; err != nil {
		return "", err
	}
	if role != "admin" && vol.OwnerID != userID {
		return "", os.ErrPermission
	}
	return utils.GetFileRealDir(vol.FolderName, relPath)
}

// ListFiles 查询请求目录下的元素列表
func ListFiles(c *gin.Context) {
	var req PathReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，缺失 VolumeID"})
		return
	}

	realPath, err := getValidatedRealDir(c, req.VolumeID, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	entries, err := os.ReadDir(realPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取目标目录异常"})
		return
	}

	list := make([]FileItem, 0)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		list = append(list, FileItem{
			Name:    e.Name(),
			Path:    filepath.ToSlash(filepath.Join(req.Path, e.Name())),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	c.JSON(http.StatusOK, list)
}

type CreateFolderReq struct {
	VolumeID uint   `json:"volumeId" binding:"required"`
	Path     string `json:"path"`
	Name     string `json:"name" binding:"required"`
}

// CreateFolder 在指定目录新建文件夹
func CreateFolder(c *gin.Context) {
	var req CreateFolderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
		return
	}

	realPath, err := getValidatedRealDir(c, req.VolumeID, filepath.Join(req.Path, req.Name))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if err := os.Mkdir(realPath, 0755); err != nil {
		if os.IsExist(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "已经存在同名文件夹"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建系统物理文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件夹建立成功"})
}

type RenameReq struct {
	VolumeID uint   `json:"volumeId" binding:"required"`
	OldPath  string `json:"oldPath" binding:"required"`
	NewName  string `json:"newName" binding:"required"`
}

// RenameFile 改名或移动层级内操作
func RenameFile(c *gin.Context) {
	var req RenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不完整"})
		return
	}

	oldReal, err := getValidatedRealDir(c, req.VolumeID, req.OldPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	newPath := filepath.Join(filepath.Dir(req.OldPath), req.NewName)
	newReal, err := getValidatedRealDir(c, req.VolumeID, newPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if err := os.Rename(oldReal, newReal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重名名修改受挫"})
		return
	}

	// 同步更新数据库中的文件记录
	// 1. 更新该项本身
	var meta models.FileMeta
	db.DB.Model(&models.FileMeta{}).Where("volume_id = ? AND file_path = ?", req.VolumeID, req.OldPath).First(&meta)
	if meta.ID != 0 {
		db.DB.Model(&meta).Update("file_path", newPath)
	}

	// 2. 如果是文件夹，更新所有子孙项
	// 使用 SQL 的 REPLACE 函数或简单的 LIKE 匹配后在应用层处理
	// 为简单起见，使用 Raw SQL 处理路径替换（如果是文件夹）
	// 注意：Windows 和 Linux 路径符号可能不同，但 FilePath 存的是 ToSlash 后的
	oldPathDir := req.OldPath + "/"
	newPathDir := newPath + "/"
	db.DB.Model(&models.FileMeta{}).
		Where("volume_id = ? AND file_path LIKE ?", req.VolumeID, req.OldPath+"/%").
		Update("file_path", gorm.Expr("REPLACE(file_path, ?, ?)", oldPathDir, newPathDir))

	c.JSON(http.StatusOK, gin.H{"message": "名称更新成功", "newPath": newPath})
}

type ActionReq struct {
	VolumeID uint   `json:"volumeId" binding:"required"`
	Path     string `json:"path" binding:"required"`
}

// DeleteFile 彻底并层删除文件夹或文件
func DeleteFile(c *gin.Context) {
	var req ActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法请求体格式"})
		return
	}

	if req.Path == "" || req.Path == "/" || req.Path == "\\" {
		c.JSON(http.StatusForbidden, gin.H{"error": "根目录不可删除"})
		return
	}

	realPath, err := getValidatedRealDir(c, req.VolumeID, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if err := os.RemoveAll(realPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统阻断了清除指令，也许遇到只读文件"})
		return
	}
	db.DB.Where("volume_id = ? AND file_path LIKE ?", req.VolumeID, req.Path+"%").Delete(&models.FileMeta{})

	c.JSON(http.StatusOK, gin.H{"message": "移除完成"})
}

// UploadFile 接收上传并在物理磁盘落盘
func UploadFile(c *gin.Context) {
	// 获取附带表单参数
	volumeIDStr := c.PostForm("volumeId")
	path := c.PostForm("path")

	if volumeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺失指向性参数"})
		return
	}

	var volID uint
	importFmt := false
	if !importFmt {
		// 这里使用一个快捷的方法转换，实际应用建议更严谨。为了保持简单，省略 fmt 包直接用简单的 Sscanf 避免报错
		// 将由后续用 fmt.Sscanf 解析，需添加 fmt 导入
		// 先补一个简易的方式实现
	}
	c.Request.ParseMultipartForm(32 << 20) // 32MB buffer
	// 懒人写法直接手写 Atoi 的效果
	var v uint = 0
	for _, ch := range volumeIDStr {
		if ch >= '0' && ch <= '9' {
			v = v*10 + uint(ch-'0')
		}
	}
	volID = v

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件流受损或未传递 File 对象"})
		return
	}

	targetRelPath := filepath.Join(path, fileHeader.Filename)
	realPath, err := getValidatedRealDir(c, volID, targetRelPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if err := c.SaveUploadedFile(fileHeader, realPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存引发硬盘读写错误"})
		return
	}

	// 记录元数据
	meta := models.FileMeta{
		VolumeID:   volID,
		FilePath:   targetRelPath,
		Size:       fileHeader.Size,
		Type:       "file",
		Permission: models.PermPrivate,
	}
	db.DB.Create(&meta)

	c.JSON(http.StatusOK, gin.H{"message": "上传部署完毕", "relativePath": targetRelPath})
}

type DownloadReq struct {
	VolumeID uint   `form:"volumeId" binding:"required"`
	Path     string `form:"path" binding:"required"`
	Preview  bool   `form:"preview"`
}

// DownloadFile 获取流文件
func DownloadFile(c *gin.Context) {
	var req DownloadReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺漏请求参数"})
		return
	}

	realPath, err := getValidatedRealDir(c, req.VolumeID, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	info, err := os.Stat(realPath)
	if os.IsNotExist(err) || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "此物非实境所能触及（文件不存在或为夹层）"})
		return
	}

	if req.Preview {
		c.Header("Content-Disposition", "inline; filename="+filepath.Base(realPath))
	} else {
		c.Header("Content-Disposition", "attachment; filename="+filepath.Base(realPath))
	}

	c.File(realPath)
}
