package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/phone_management/configs"
)

// Claims 定义了JWT中存储的自定义声明。
// JTI (ID) 会通过内嵌的 jwt.RegisteredClaims 提供
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

var (
	// tokenDenylist 存储已登出Token的JTI及其原始过期时间。
	// key: JTI (JWT ID), value: 该JTI的原始过期时间点。
	// 注意: 这是一个内存列表，服务重启会丢失。生产环境应使用Redis等持久化存储。
	tokenDenylist = make(map[string]time.Time)
	denylistMutex = &sync.RWMutex{}
)

// AddToDenylist 将JTI添加到拒绝列表，并清理已过期的条目。
func AddToDenylist(jti string, expiresAt time.Time) {
	denylistMutex.Lock()
	defer denylistMutex.Unlock()

	tokenDenylist[jti] = expiresAt

	// 清理拒绝列表中其他已完全过期的JTI
	now := time.Now()
	for id, exp := range tokenDenylist {
		if now.After(exp) {
			delete(tokenDenylist, id)
		}
	}
}

// IsTokenDenylisted 检查JTI是否在拒绝列表中且尚未过期。
func IsTokenDenylisted(jti string) bool {
	denylistMutex.RLock()
	defer denylistMutex.RUnlock()

	expTime, found := tokenDenylist[jti]
	if !found {
		return false // 不在拒绝列表
	}

	// 如果JTI在拒绝列表中，且其记录的过期时间点仍在未来，则认为是 فعال (denylisted)
	return time.Now().Before(expTime)
}

// JWTMiddleware 是一个Gin中间件，用于验证JWT。
// 它从 Authorization 请求头中提取 Bearer Token，
// 并使用 `golang-jwt/jwt/v5` 库进行验证。
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 确保token的签名方法是我们期望的 HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(configs.AppConfig.JWTSecret), nil // 使用配置中的密钥
		})

		if err != nil {
			// 使用 errors.Is 来判断特定的JWT错误类型
			if errors.Is(err, jwt.ErrTokenMalformed) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is malformed"})
			} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is expired or not valid yet"})
			} else if errors.Is(err, jwt.ErrSignatureInvalid) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token signature"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			}
			c.Abort()
			return
		}

		if !token.Valid { // ParseWithClaims 验证失败时 err != nil，此检查可能多余但无害
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is invalid"})
			c.Abort()
			return
		}

		// 检查JTI是否存在
		if claims.ID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing JTI (JWT ID)"})
			c.Abort()
			return
		}

		// 检查Token是否已在拒绝列表
		if IsTokenDenylisted(claims.ID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been invalidated (logged out)"})
			c.Abort()
			return
		}

		// 将声明和关键信息存储在Gin上下文中，以便后续处理程序使用
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("jti", claims.ID) // 存储JTI
		if claims.ExpiresAt != nil {
			c.Set("exp", claims.ExpiresAt.Time) // 存储过期时间
		}

		c.Next()
	}
}
