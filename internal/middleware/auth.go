package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/DenisChesnokov/go-shortener.git/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// CookieAuth — middleware для аутентификации через JWT-куку.
// Если куки нет — создаёт нового пользователя (UUID).
// Если кука невалидна — возвращает 401.
// Если всё ок — кладёт userID в context.
func CookieAuth(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("user_id")
			if err != nil {
				// Куки нет — генерируем нового пользователя
				userID := uuid.New().String()

				tokenString, err := jwtMgr.BuildJWTString(userID)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "user_id",
					Value:    tokenString,
					Path:     "/",
					HttpOnly: true,
				})

				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Кука есть — парсим
			userID, err := jwtMgr.ParseToken(cookie.Value)
			if err != nil {
				// Невалидная/просроченная кука — генерируем нового пользователя
				userID = uuid.New().String()
			}

			// Ставим куку (перевыпускаем при любом раскладе — и при ошибке и при успехе)
			tokenString, err := jwtMgr.BuildJWTString(userID)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "user_id",
				Value:    tokenString,
				Path:     "/",
				HttpOnly: true,
			})

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает userID из context.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
