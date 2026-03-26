package middlewares

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"os"
)

var JwtSecret = []byte(getEnv("JWT_SECRET", "mypan_super_secret_key"))

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GenerateToken 生成具有指定有效期的 JWT Token
func GenerateToken(userID uint, username, role string, expSeconds int64) (string, error) {
	claims := jwt.MapClaims{
		"userID":   userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Duration(expSeconds) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

// VerifyToken 验证 token 并返回 claims
func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return JwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

// AuthMiddleware HTTP 拦截器：解析 JWT，防止盗链或越权
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先读 Authorization header，其次兼容 query param ?token= （用于浏览器直接下载场景）
		authHeader := c.GetHeader("Authorization")
		var tokenString string
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式错误，需采用 Bearer [Token]"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else if t := c.Query("token"); t != "" {
			tokenString = t
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺失 Authorization Header 或 token 参数"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			log.Println("Token解析异常或已过期:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 Token 或已过期，请重新登录"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 Token Payload"})
			c.Abort()
			return
		}

		// 把信息写入当前 Gin 的 Context，供 Controller 层利用
		c.Set("userID", uint(claims["userID"].(float64)))
		c.Set("username", claims["username"].(string))
		c.Set("role", claims["role"].(string))

		c.Next()
	}
}
