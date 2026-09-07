package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

type JWTManager struct {
	secret []byte
	exp    time.Duration
}

func NewJWTManager(secret string, exp time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		exp:    exp,
	}
}

// BuildJWTString создаёт подписанный JWT-токен для userID.
func (m *JWTManager) BuildJWTString(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.exp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ParseToken проверяет подпись и возвращает userID из токена.
func (m *JWTManager) ParseToken(tokenString string) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		})
	if err != nil {
		return "", ErrInvalidToken
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	return claims.UserID, nil
}
