package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/phone_management/internal/auth"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/pkg/db"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// JWT密钥，应从配置中读取
var jwtKey = []byte("mobile") // TODO: 从配置管理中获取密钥

// Login 处理管理员登录请求
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "details": err.Error()})
		return
	}

	var user models.User
	if err := db.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户名或密码"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户名或密码"})
		return
	}

	// 生成JWT
	expirationTime := time.Now().Add(24 * time.Hour) // Token 有效期24小时
	claims := &jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		Subject:   user.Username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Issuer:    "phone_system",            // 可选，签发者
		Audience:  jwt.ClaimStrings{"admin"}, // 可选，受众
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成Token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: tokenString,
		User: UserInfo{
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// LogoutHandler godoc
// @Summary User logout
// @Description Logs out the current user by invalidating their token.
// @Tags auth
// @Security BearerAuth
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string "Logout successful"
// @Failure 400 {object} map[string]string "Bad Request (e.g., missing JTI or EXP in context)"
// @Failure 401 {object} map[string]string "Unauthorized (token issues handled by JWTMiddleware)"
// @Router /auth/logout [post]
func LogoutHandler(c *gin.Context) {
	jtiVal, jtiExists := c.Get("jti")
	expVal, expExists := c.Get("exp")

	if !jtiExists || !expExists {
		// 这通常不应该发生，因为JWTMiddleware应该已经填充了这些值
		// 或者如果logout路由没有被JWTMiddleware正确保护
		c.JSON(http.StatusBadRequest, gin.H{"error": "Logout context error: JTI or EXP not found in context"})
		return
	}

	jti, okJTI := jtiVal.(string)
	exp, okEXP := expVal.(time.Time)

	if !okJTI || jti == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Logout context error: Invalid JTI"})
		return
	}
	if !okEXP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Logout context error: Invalid EXP"})
		return
	}

	auth.AddToDenylist(jti, exp) // 将JTI添加到拒绝列表
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}
