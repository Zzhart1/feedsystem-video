package jwt

import (
	"errors"
	"net/http"
	"strings"

	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/auth"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"github.com/gin-gonic/gin"
)

// JWTAuth check jwt token and ensure it matches the currently stored token.
func JWTAuth(accountRepo *account.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokenString := parts[1]
		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		check(c, claims, tokenString, accountRepo, cache)
	}
}

func SoftJWTAuth(accountRepo *account.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authAuthorization := c.GetHeader("Authorization")
		if authAuthorization == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(authAuthorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		tokenStr := parts[1]
		claim, err := auth.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		check(c, claim, tokenStr, accountRepo, cache)
	}
}

// check 校验 token 与 DB 中存储的 token 一致（服务端可撤销），并注入登录态。
// 当前直接查 DB 兜底；阶段 2 接入 Redis 后可先查缓存、miss 再回源 DB。
func check(c *gin.Context, claims *auth.Claims, tokenString string, accountRepo *account.AccountRepository, cache *rediscache.Client) {
	acc, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
	if err != nil || acc.Token != tokenString {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token mismatch or account not found"})
		return
	}
	c.Set("accountID", acc.ID)
	c.Set("username", acc.Username)
	c.Next()
}

func GetAccountID(c *gin.Context) (uint, error) {
	accountID, exists := c.Get("accountID")
	if !exists {
		return 0, errors.New("account not founded")
	}
	id, ok := accountID.(uint)
	if !ok {
		return 0, errors.New("accountID has invalid type")
	}
	return id, nil
}

func GetUsername(c *gin.Context) (string, error) {
	username, exists := c.Get("username")
	if !exists {
		return "", errors.New("username not founded")
	}
	name, ok := username.(string)
	if !ok {
		return "", errors.New("username has invalid type")
	}
	return name, nil
}
