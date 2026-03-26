package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"mypan-backend/db"
	"mypan-backend/middlewares"
	"mypan-backend/models"
)

type UserRequest struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	DurationSeconds int64  `json:"durationSeconds"` // 可选，由前端决定 Cookie 和 Token 的生命周期
}

// Status 返回系统是否已完成初始化（是否已有账号）
func Status(c *gin.Context) {
	var count int64
	db.DB.Model(&models.User{}).Count(&count)
	c.JSON(http.StatusOK, gin.H{"initialized": count > 0})
}

// Register 用户端注册接口（仅允许系统首次注册，之后禁止任何新账号创建）
func Register(c *gin.Context) {
	// 前置检查：系统已有账号时，禁止一切注册行为
	var totalUsers int64
	db.DB.Model(&models.User{}).Count(&totalUsers)
	if totalUsers > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统已完成初始化，禁止创建新账号"})
		return
	}

	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号和密码不能为空"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器处理密码时遇到了阻碍"})
		return
	}

	// 第一个也是唯一的账号，自动赋予 admin 权限
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         "admin",
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立档案失败，数据库异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功！系统管理员账号已建立。",
		"role":    "admin",
	})
}

// Login 会返回 JWT Token
func Login(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号和密码不能为空"})
		return
	}

	var user models.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "抱歉，找不到该用户或凭证错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码不匹配，核实后再试"})
		return
	}

	// 确定有效时长，默认为 7 天 (3600*24*7)
	duration := req.DurationSeconds
	if duration <= 0 {
		duration = 3600 * 24 * 7
	}

	token, err := middlewares.GenerateToken(user.ID, user.Username, user.Role, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "门禁卡颁发故障了，Token生成失败"})
		return
	}

	// 为 HTML 静态页访问回填 Cookie，时效与 JWT 同步
	c.SetCookie("mypan_token", token, int(duration), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
