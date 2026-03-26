package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"mypan-backend/db"
	"mypan-backend/middlewares"
	"mypan-backend/models"
	"mypan-backend/utils"
)

const publicCookieName = "mypan_vol_access"

// PublicVolumeIndex 处理 /s/:key 请求，根据卷的访问模式决定行为
func PublicVolumeIndex(c *gin.Context) {
	key := c.Param("key")

	var vol models.Volume
	if err := db.DB.Where("access_url_key = ?", key).First(&vol).Error; err != nil {
		c.String(http.StatusNotFound, "404 - 访问链接不存在或已失效")
		return
	}

	if vol.AccessMode == models.VolumeAccessPrivate {
		c.String(http.StatusForbidden, "403 - 此卷为私有存储，无权访问")
		return
	}

	// 密码模式：检查 cookie 或让用户提交密码
	if vol.AccessMode == models.VolumeAccessPassword {
		cookie, err := c.Cookie(publicCookieName + "_" + key)
		if err != nil || cookie != "ok" {
			// 如果是 POST，验证密码
			if c.Request.Method == "POST" {
				if !utils.SharePasswordLimiter.Allow(c.ClientIP()) {
					renderPasswordPage(c, key, "尝试次数过多，请稍后再访问")
					return
				}
				pwd := c.PostForm("password")
				if bcrypt.CompareHashAndPassword([]byte(vol.AccessPassword), []byte(pwd)) == nil {
					c.SetCookie(publicCookieName+"_"+key, "ok", 3600*24, "/", "", false, true)
					c.Redirect(http.StatusFound, "/s/"+key)
					return
				}
				renderPasswordPage(c, key, "密码错误，请重试")
				return
			}
			renderPasswordPage(c, key, "")
			return
		}
	}

	// login 模式：检查 JWT token cookie（mypan_token）
	if vol.AccessMode == models.VolumeAccessLogin {
		token, err := c.Cookie("mypan_token")
		if err != nil || token == "" {
			// 如果是公开访问且未登录，重定向到公共登录页
			c.Redirect(http.StatusFound, "/login?redirect=/s/"+key)
			return
		}
	}

	// 渲染文件列表
	relPath := c.Query("path")
	renderFileList(c, vol, relPath)
}

// PublicVolumeDownload 处理公开访问的文件下载 /s/:key/download?path=xxx
func PublicVolumeDownload(c *gin.Context) {
	key := c.Param("key")
	relPath := c.Query("path")

	if relPath == "" {
		c.String(http.StatusBadRequest, "缺少 path 参数")
		return
	}

	var vol models.Volume
	if err := db.DB.Where("access_url_key = ?", key).First(&vol).Error; err != nil {
		c.String(http.StatusNotFound, "卷不存在")
		return
	}

	if vol.AccessMode == models.VolumeAccessPrivate {
		c.String(http.StatusForbidden, "私有卷")
		return
	}

	if vol.AccessMode == models.VolumeAccessPassword {
		cookie, err := c.Cookie(publicCookieName + "_" + key)
		if err != nil || cookie != "ok" {
			c.String(http.StatusUnauthorized, "需要密码验证")
			return
		}
	}

	if vol.AccessMode == models.VolumeAccessLogin {
		token, err := c.Cookie("mypan_token")
		if err != nil || token == "" {
			c.String(http.StatusUnauthorized, "需要登录")
			return
		}
	}

	realPath, err := utils.GetFileRealDir(vol.FolderName, relPath)
	if err != nil {
		c.String(http.StatusBadRequest, "非法路径")
		return
	}

	info, err := os.Stat(realPath)
	if os.IsNotExist(err) || info.IsDir() {
		c.String(http.StatusNotFound, "文件不存在")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(realPath)+"\"")
	
	// 处理预览逻辑 (Security Analysis: inline vs attachment)
	if c.Query("preview") == "1" {
		ext := strings.ToLower(filepath.Ext(realPath))
		// 1. 危险文件强制下载 (Prevention of arbitrary code execution)
		dangerousExts := map[string]bool{".exe": true, ".sh": true, ".apk": true, ".bat": true, ".msi": true}
		if !dangerousExts[ext] {
			c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(realPath)+"\"")
			// 2. 潜在 XSS 攻击防护 (Inject CSP for HTML/SVG)
			if ext == ".html" || ext == ".htm" || ext == ".svg" {
				c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:;")
			}
		}
	}

	c.File(realPath)
}

const passwordPageTmpl = `<!DOCTYPE html><html lang="zh"><head><meta charset="UTF-8">
<title>访问受保护存储卷</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f3f4f6}
.box{background:#fff;padding:32px;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.08);min-width:300px;text-align:center}
h2{margin:0 0 16px;font-size:1.2rem;color:#1f2937}
input{width:100%;padding:10px;border:1px solid #d1d5db;border-radius:8px;font-size:1rem;box-sizing:border-box;margin-bottom:12px}
button{width:100%;padding:10px;background:#4f46e5;color:#fff;border:none;border-radius:8px;font-size:1rem;cursor:pointer}
button:hover{background:#4338ca}</style></head>
<body><div class="box">
<h2>🔒 需要访问密码</h2>{{if .Error}}<p style="color:red;margin:8px 0">{{.Error}}</p>{{end}}
<form method="POST"><input type="password" name="password" placeholder="请输入访问密码" autofocus required><button type="submit">确认访问</button></form>
</div></body></html>`

func renderPasswordPage(c *gin.Context, key, errMsg string) {
	tmpl := template.Must(template.New("pwd").Parse(passwordPageTmpl))
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(c.Writer, map[string]string{"Error": errMsg})
}

const fileListTmpl = `<!DOCTYPE html><html lang="zh"><head><meta charset="UTF-8">
<title>{{.VolName}} - MyPan 共享卷</title>
<style>
	body{font-family:sans-serif;max-width:900px;margin:40px auto;padding:0 20px;color:#1f2937;background:#f9fafb}
	.container{background:#fff;padding:32px;border-radius:12px;box-shadow:0 1px 3px rgba(0,0,0,.1)}
	h1{font-size:1.5rem;margin:0 0 20px 0;color:#111827;border-bottom:1px solid #eee;padding-bottom:15px}
	.breadcrumb{color:#6b7280;font-size:0.95rem;margin-bottom:24px;background:#f3f4f6;padding:10px 16px;border-radius:8px;display:flex;align-items:center}
	.breadcrumb a{color:#4f46e5;text-decoration:none;font-weight:500}
	.breadcrumb span{margin:0 8px;color:#d1d5db;font-weight:bold}
	table{width:100%;border-collapse:collapse;margin-top:10px}
	th{text-align:left;padding:12px 10px;background:#f9fafb;border-bottom:2px solid #e5e7eb;font-size:0.85rem;color:#6b7280;text-transform:uppercase}
	td{padding:14px 10px;border-bottom:1px solid #f3f4f6;font-size:0.95rem}
	tr:hover td{background:#fafafa}
	.empty{text-align:center;padding:60px;color:#9ca3af;font-style:italic}
    .btn-download{color:#4f46e5;text-decoration:none}
</style></head>
<body><div class="container">
	<h1>📦 {{.VolName}}</h1>
	<div class="breadcrumb">{{.Breadcrumb}}</div>
	<table><thead><tr><th>名称</th><th>大小</th><th>修改时间</th><th>操作</th></tr></thead>
	<tbody>{{range .Files}}
		<tr>
			<td>{{if .IsDir}}📁 <a href="/s/{{$.Key}}?path={{.RelPath}}">{{.Name}}</a>{{else}}📄 <a href="/s/{{$.Key}}/download?path={{.RelPath}}&preview=1" target="_blank">{{.Name}}</a>{{end}}</td>
			<td>{{.Size}}</td><td>{{.ModTime}}</td><td>{{if not .IsDir}}<a href="/s/{{$.Key}}/download?path={{.RelPath}}" class="btn-download">下载</a>{{else}}-{{end}}</td>
		</tr>
	{{else}}<tr><td colspan="4" class="empty">此目录为空</td></tr>{{end}}
	</tbody></table>
</div></body></html>`

type fileEntry struct {
	Name    string
	RelPath string
	IsDir   bool
	Size    string
	ModTime string
}

func renderFileList(c *gin.Context, vol models.Volume, relPath string) {
	realDir, err := utils.GetFileRealDir(vol.FolderName, relPath)
	if err != nil {
		c.String(http.StatusBadRequest, "非法路径")
		return
	}
	entries, _ := os.ReadDir(realDir)
	key := utils.PtrToString(vol.AccessURLKey)

	// 面包屑生成
	breadcrumbHTML := template.HTML(fmt.Sprintf(`<a href="/s/%s">全部文件</a>`, key))
	if relPath != "" {
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		accumulatedPath := ""
		for _, part := range parts {
			if part == "" { continue }
			if accumulatedPath == "" { accumulatedPath = part } else { accumulatedPath += "/" + part }
			breadcrumbHTML += template.HTML(fmt.Sprintf(` <span>&rsaquo;</span> <a href="/s/%s?path=%s">%s</a>`, key, accumulatedPath, part))
		}
	}

	files := make([]fileEntry, 0)
	for _, e := range entries {
		info, _ := e.Info()
		files = append(files, fileEntry{
			Name:    e.Name(),
			RelPath: filepath.ToSlash(filepath.Join(relPath, e.Name())),
			IsDir:   e.IsDir(),
			Size:    utils.HumanSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}

	tmpl := template.Must(template.New("list").Parse(fileListTmpl))
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(c.Writer, map[string]interface{}{
		"VolName":    vol.Name,
		"Breadcrumb": breadcrumbHTML,
		"Files":      files,
		"Key":        key,
	})
}

// 下面这行仅用于消除 time 包的 import 未使用警告（在渲染中 ModTime 已使用）
// PublicLogin 处理公共登录页 GET/POST
func PublicLogin(c *gin.Context) {
	redirect := c.DefaultQuery("redirect", "/")
	if !utils.IsSafeRedirect(redirect) {
		redirect = "/"
	}

	if c.Request.Method == "GET" {
		renderPublicLoginPage(c, "", redirect)
		return
	}

	// POST 处理
	if !utils.LoginLimiter.Allow(c.ClientIP()) {
		renderPublicLoginPage(c, "尝试登录次数过多，请稍后再试", redirect)
		return
	}
	username := c.PostForm("username")
	password := c.PostForm("password")
	durationStr := c.DefaultPostForm("duration", "604800") // 默认 7 天
	duration, _ := strconv.ParseInt(durationStr, 10, 64)

	var user models.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		renderPublicLoginPage(c, "账号不存在", redirect)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		renderPublicLoginPage(c, "密码错误", redirect)
		return
	}

	token, _ := middlewares.GenerateToken(user.ID, user.Username, user.Role, duration)
	c.SetCookie("mypan_token", token, int(duration), "/", "", false, true)
	
	// 重定向安全校验：确保 redirect 依然安全
	if !utils.IsSafeRedirect(redirect) {
		redirect = "/"
	}
	c.Redirect(http.StatusFound, redirect)
}

const volLoginPageTmpl = `<!DOCTYPE html><html lang="zh"><head><meta charset="UTF-8">
<title>登录 MyPan</title>
<style>
	body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f3f4f6}
	.box{background:#fff;padding:40px;border-radius:16px;box-shadow:0 10px 25px rgba(0,0,0,.05);width:100%;max-width:360px}
	h2{margin:0 0 8px;font-size:1.5rem;color:#111827;text-align:center}
	p{margin:0 0 32px;color:#6b7280;font-size:0.9rem;text-align:center}
	.field{margin-bottom:20px}
	label{display:block;margin-bottom:6px;font-size:0.85rem;color:#374151;font-weight:500}
	input, select{width:100%;padding:12px;border:1px solid #d1d5db;border-radius:8px;font-size:1rem;box-sizing:border-box}
	button{width:100%;padding:12px;background:#4f46e5;color:#fff;border:none;border-radius:8px;font-size:1rem;font-weight:600;cursor:pointer;margin-top:10px}
</style></head>
<body><div class="box">
	<h2>欢迎回来</h2><p>请输入凭证以访问受限资源</p>
	{{if .Error}}<div style="color:#ef4444;background:#fee2e2;padding:12px;border-radius:8px;margin-bottom:16px;font-size:0.9rem">{{.Error}}</div>{{end}}
	<form method="POST">
		<div class="field"><label>用户名</label><input type="text" name="username" placeholder="输入用户名" required autofocus></div>
		<div class="field"><label>密码</label><input type="password" name="password" placeholder="输入密码" required></div>
		<div class="field"><label>保持登录时长</label>
			<select name="duration"><option value="3600">1 小时</option><option value="86400">1 天</option><option value="604800" selected>1 周</option><option value="31536000">1 年</option></select>
		</div>
		<input type="hidden" name="redirect" value="{{.Redirect}}">
		<button type="submit">立即登录</button>
	</form>
</div></body></html>`

func renderPublicLoginPage(c *gin.Context, errMsg, redirect string) {
	tmpl := template.Must(template.New("vol-login").Parse(volLoginPageTmpl))
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(c.Writer, map[string]string{"Error": errMsg, "Redirect": redirect})
}

var _ = time.Now
