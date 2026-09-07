package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DenisChesnokov/go-shortener.git/internal/auth"
)

func TestCookieAuth_NoCookie(t *testing.T) {
	mgr := auth.NewJWTManager("secret", time.Hour)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			t.Error("userID не в context")
		}
		if userID == "" {
			t.Error("userID пустой")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := CookieAuth(mgr)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: получили %d, хотим %d", rec.Code, http.StatusOK)
	}

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "user_id" {
			found = true
		}
	}
	if !found {
		t.Error("cookie user_id не установлена")
	}
}

func TestCookieAuth_ValidCookie(t *testing.T) {
	mgr := auth.NewJWTManager("secret", time.Hour)
	token, _ := mgr.BuildJWTString("user-xyz")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := CookieAuth(mgr)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: получили %d, хотим %d", rec.Code, http.StatusOK)
	}
	if gotUserID != "user-xyz" {
		t.Errorf("userID: получили %q, хотим %q", gotUserID, "user-xyz")
	}
}

func TestCookieAuth_InvalidCookie(t *testing.T) {
	mgr := auth.NewJWTManager("secret", time.Hour)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := CookieAuth(mgr)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "bad-token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: получили %d, хотим %d", rec.Code, http.StatusOK)
	}
	if gotUserID == "" {
		t.Error("userID не должен быть пустым")
	}

	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "user_id" {
			found = true
		}
	}
	if !found {
		t.Error("cookie user_id должна быть установлена")
	}
}
