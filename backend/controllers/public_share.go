package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"mypan-backend/middlewares"
	"mypan-backend/db"
	"mypan-backend/models"
	"mypan-backend/utils"
)

const publicFileCookieName = "mypan_file_access"

// PublicFileIndex 处理 /f/:key 请求
func PublicFileIndex(c *gin.Context) {
	key := c.Param("key")

	var meta models.FileMeta
	if err := db.DB.Where("access_url_key = ?", key).First(&meta).Error; err != nil {
		c.String(http.StatusNotFound, "404 - 分享链接不存在或已失效")
		return
	}

	// 检查是否过期
	if meta.ExpiresAt != nil && time.Now().After(*meta.ExpiresAt) {
		c.String(http.StatusGone, "410 - 该分享链接已过期")
		return
	}

	if meta.Permission == models.PermPrivate {
		c.String(http.StatusForbidden, "403 - 该分享已被撤销")
		return
	}

	// 密码模式
	if meta.Permission == models.PermPassword {
		cookie, err := c.Cookie(publicFileCookieName + "_" + key)
		if err != nil || cookie != "ok" {
			if c.Request.Method == "POST" {
				pwd := c.PostForm("password")
				if bcrypt.CompareHashAndPassword([]byte(meta.PasswordHash), []byte(pwd)) == nil {
					c.SetCookie(publicFileCookieName+"_"+key, "ok", 3600*24, "/", "", false, true)
					c.Redirect(http.StatusFound, "/f/"+key)
					return
				}
				renderFilePasswordPage(c, key, "密码错误")
				return
			}
			renderFilePasswordPage(c, key, "")
			return
		}
	}

	// 仅登录模式
	if meta.Permission == models.PermLogin {
		// 检查是否有有效 token (头部或 Cookie)
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			cookie, _ := c.Cookie("mypan_token")
			tokenStr = cookie
		} else {
			tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		}

		if tokenStr == "" {
			renderLoginPage(c, "此分享仅限登录用户查看，请先登录")
			return
		}

		_, err := middlewares.VerifyToken(tokenStr)
		if err != nil {
			renderLoginPage(c, "登录已失效，请重新登录后再访问")
			return
		}
	}

	// 获取卷信息
	var vol models.Volume
	db.DB.First(&vol, meta.VolumeID)

	if meta.Type == "directory" {
		relPath := c.Query("path")
		// 最终物理路径是 meta.FilePath + relPath
		fullRelPath := filepath.ToSlash(filepath.Join(meta.FilePath, relPath))
		renderSharedFileList(c, vol, meta, fullRelPath, relPath)
	} else {
		renderSharedFileInfo(c, vol, meta)
	}
}

// PublicFileDownload 处理 /f/:key/download
func PublicFileDownload(c *gin.Context) {
	key := c.Param("key")
	subPath := c.Query("path")

	var meta models.FileMeta
	if err := db.DB.Where("access_url_key = ?", key).First(&meta).Error; err != nil {
		c.String(http.StatusNotFound, "分享不存在")
		return
	}

	// 校验过期和权限 (同 Index)
	if meta.ExpiresAt != nil && time.Now().After(*meta.ExpiresAt) {
		c.String(http.StatusGone, "链接已过期")
		return
	}
	if meta.Permission == models.PermPassword {
		cookie, err := c.Cookie(publicFileCookieName + "_" + key)
		if err != nil || cookie != "ok" {
			c.String(http.StatusUnauthorized, "需要密码")
			return
		}
	}

	if meta.Permission == models.PermLogin {
		tokenStr, _ := c.Cookie("mypan_token")
		if t := c.Query("token"); t != "" {
			tokenStr = t
		}
		if _, err := middlewares.VerifyToken(tokenStr); err != nil {
			c.String(http.StatusUnauthorized, "需要有效登录")
			return
		}
	}

	var vol models.Volume
	db.DB.First(&vol, meta.VolumeID)

	targetRelPath := meta.FilePath
	if meta.Type == "directory" && subPath != "" {
		targetRelPath = filepath.Join(meta.FilePath, subPath)
	}

	realPath, err := utils.GetFileRealDir(vol.FolderName, targetRelPath)
	if err != nil {
		c.String(http.StatusBadRequest, "路径异常")
		return
	}

	info, err := os.Stat(realPath)
	if err != nil || info.IsDir() {
		c.String(http.StatusNotFound, "资源不可用")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(realPath)+"\"")
	if c.Query("preview") == "1" {
		c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(realPath)+"\"")
	}
	c.File(realPath)
}

func renderFilePasswordPage(c *gin.Context, key, errMsg string) {
	errHTML := ""
	if errMsg != "" {
		errHTML = fmt.Sprintf(`<p style="color:red">%s</p>`, errMsg)
	}
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>输入提取码</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;background:#f3f4f6}
.card{background:#fff;padding:40px;border-radius:12px;box-shadow:0 4px 6px rgba(0,0,0,0.1);text-align:center}</style></head>
<body><div class="card"><h2>🔐 此分享受密码保护</h2>%s<form method="POST"><input type="password" name="password" placeholder="请输入密码" style="padding:10px;border-radius:6px;border:1px solid #ddd;width:200px"><button type="submit" style="padding:10px 20px;background:#4f46e5;color:#fff;border:none;border-radius:6px;margin-left:8px;cursor:pointer">访问</button></form></div></body></html>`, errHTML)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func renderLoginPage(c *gin.Context, msg string) {
	// 获取当前完整路径以便登录后回跳
	currentPath := c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		currentPath += "?" + c.Request.URL.RawQuery
	}
	loginUrl := "/login?redirect=" + url.QueryEscape(currentPath)

	// 简单的 HTML 提示并展示登录按钮
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>需要登录</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;background:#f3f4f6}
.card{background:#fff;padding:40px;border-radius:12px;box-shadow:0 4px 6px rgba(0,0,0,0.1);text-align:center}</style></head>
<body><div class="card"><h2>🔒 访问受限</h2><p style="color:#666">%s</p><a href="%s" style="padding:10px 20px;background:#4f46e5;color:#fff;text-decoration:none;border-radius:6px;display:inline-block;margin-top:16px">前往登录</a></div></body></html>`, msg, loginUrl)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func renderSharedFileInfo(c *gin.Context, vol models.Volume, meta models.FileMeta) {
	name := filepath.Base(meta.FilePath)
	size := humanSize(meta.Size)
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>%s - MyPan 分享</title>
<style>body{font-family:sans-serif;background:#f9fafb;display:flex;justify-content:center;padding-top:100px}
.box{background:#fff;padding:40px;border-radius:12px;box-shadow:0 1px 3px rgba(0,0,0,0.1);max-width:500px;width:100%%;text-align:center}
h1{font-size:1.25rem;margin-bottom:8px}
.info{color:#6b7280;margin-bottom:24px}
.btn{display:inline-block;padding:12px 24px;background:#4f46e5;color:#fff;text-decoration:none;border-radius:8px;font-weight:500}</style></head>
<body><div class="box"><h1>📄 %s</h1><div class="info">大小: %s</div><a href="/f/%s/download" class="btn">立即下载</a><a href="/f/%s/download?preview=1" style="margin-left:12px;color:#4f46e5" target="_blank">预览</a></div></body></html>`,
		name, name, size, meta.AccessURLKey, meta.AccessURLKey)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func renderSharedFileList(c *gin.Context, vol models.Volume, meta models.FileMeta, fullRelPath, subPath string) {
	realDir, _ := utils.GetFileRealDir(vol.FolderName, fullRelPath)
	entries, _ := os.ReadDir(realDir)

	rows := ""
	for _, e := range entries {
		info, _ := e.Info()
		name := e.Name()
		currentSubPath := filepath.ToSlash(filepath.Join(subPath, name))

		if e.IsDir() {
			rows += fmt.Sprintf(`<tr><td>📁 <a href="/f/%s?path=%s">%s</a></td><td>-</td><td>-</td></tr>`,
				meta.AccessURLKey, currentSubPath, name)
		} else {
			rows += fmt.Sprintf(`<tr><td>📄 %s</td><td>%s</td><td><a href="/f/%s/download?path=%s" style="color:#4f46e5">下载</a></td></tr>`,
				name, humanSize(info.Size()), meta.AccessURLKey, currentSubPath)
		}
	}

	breadcrumb := fmt.Sprintf(`<a href="/f/%s">根目录</a>`, meta.AccessURLKey)
	if subPath != "" {
		parts := strings.Split(subPath, "/")
		acc := ""
		for _, p := range parts {
			if p == "" { continue }
			if acc == "" { acc = p } else { acc += "/" + p }
			breadcrumb += fmt.Sprintf(` <span>&rsaquo;</span> <a href="/f/%s?path=%s">%s</a>`, meta.AccessURLKey, acc, p)
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>文件夹分享 - MyPan</title>
<style>body{font-family:sans-serif;max-width:800px;margin:40px auto;background:#f3f4f6}
.card{background:#fff;padding:32px;border-radius:12px;box-shadow:0 1px 3px rgba(0,0,0,0.1)}
.breadcrumb{margin-bottom:20px;color:#6b7280}
.breadcrumb a{color:#4f46e5;text-decoration:none}
table{width:100%%;border-collapse:collapse}
th{text-align:left;padding:12px;border-bottom:2px solid #eee}
td{padding:12px;border-bottom:1px solid #f9f9f9}
</style></head>
<body><div class="card"><h2>📁 文件夹分享: %s</h2><div class="breadcrumb">%s</div>
<table><thead><tr><th>名称</th><th>大小</th><th>操作</th></tr></thead><tbody>%s</tbody></table></div></body></html>`,
		filepath.Base(meta.FilePath), breadcrumb, rows)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
