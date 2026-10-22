package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AccountID uint   `json:"account_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("feedsystem-dev-secret-key")
}

func GenerateToken(accountID uint, username string) (string, error) {
	return sign(accountID, username, 15*time.Minute)
}

func GenerateRefreshToken(accountID uint, username string) (string, error) {
	return sign(accountID, username, 7*24*time.Hour)
}

func sign(accountID uint, username string, ttl time.Duration) (string, error) {
	claims := &Claims{
		AccountID: accountID,
		Username:  username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return secret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invaid token")
	}
	return claims, nil
}
