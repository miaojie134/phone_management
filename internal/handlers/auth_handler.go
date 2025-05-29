package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/phone_management/configs"
	"github.com/phone_management/internal/auth"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/pkg/db"
	"github.com/phone_management/pkg/utils"
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

// Login godoc
// @Summary 管理员登录
// @Description 验证管理员凭证并返回 JWT
// @Tags auth
// @Accept  json
// @Produce  json
// @Param credentials body LoginRequest true "登录凭证"
// @Success 200 {object} utils.SuccessResponse{data=LoginResponse} "登录成功，返回 Token 和用户信息"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 401 {object} utils.APIErrorResponse "无效的用户名或密码"
// @Failure 500 {object} utils.APIErrorResponse "无法生成Token"
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	var user models.User
	if err := db.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		utils.RespondUnauthorizedError(c, "无效的用户名或密码")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.RespondUnauthorizedError(c, "无效的用户名或密码")
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &auth.Claims{
		UserID:   uint(user.ID),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   user.Username,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "phone_system",            // 可选，签发者
			Audience:  jwt.ClaimStrings{"admin"}, // 可选，受众
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(configs.AppConfig.JWTSecret))
	if err != nil {
		utils.RespondInternalServerError(c, "无法生成Token", err.Error())
		return
	}

	loginResp := LoginResponse{
		Token: tokenString,
		User: UserInfo{
			Username: user.Username,
			Role:     user.Role,
		},
	}
	utils.RespondSuccess(c, http.StatusOK, loginResp, "登录成功")
}

// LogoutHandler godoc
// @Summary User logout
// @Description Logs out the current user by invalidating their token.
// @Tags auth
// @Security BearerAuth
// @Accept  json
// @Produce  json
// @Success 200 {object} utils.SuccessResponse "成功登出"
// @Failure 400 {object} utils.APIErrorResponse "错误的请求 (例如，上下文中缺少JTI或EXP)"
// @Router /auth/logout [post]
func LogoutHandler(c *gin.Context) {
	jtiVal, jtiExists := c.Get("jti")
	expVal, expExists := c.Get("exp")

	if !jtiExists || !expExists {
		utils.RespondAPIError(c, http.StatusBadRequest, "Logout context error: JTI or EXP not found in context", nil)
		return
	}

	jti, okJTI := jtiVal.(string)
	exp, okEXP := expVal.(time.Time)

	if !okJTI || jti == "" {
		utils.RespondAPIError(c, http.StatusBadRequest, "Logout context error: Invalid JTI", nil)
		return
	}
	if !okEXP {
		utils.RespondAPIError(c, http.StatusBadRequest, "Logout context error: Invalid EXP", nil)
		return
	}

	auth.AddToDenylist(jti, exp)
	utils.RespondSuccess(c, http.StatusOK, nil, "成功登出")
}
