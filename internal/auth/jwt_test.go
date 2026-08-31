package auth

import (
	"testing"
	"time"
)

func TestBuildAndParseJWTString(t *testing.T) {
	mgr := NewJWTManager("test-secret", time.Hour)
	userID := "user-123"

	token, err := mgr.BuildJWTString(userID)
	if err != nil {
		t.Fatalf("BuildJWTString: %v", err)
	}
	if token == "" {
		t.Error("token пустой")
	}

	gotUserID, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("userID: получили %q, хотим %q", gotUserID, userID)
	}
}

func TestParseToken_InvalidString(t *testing.T) {
	mgr := NewJWTManager("test-secret", time.Hour)

	_, err := mgr.ParseToken("not-a-jwt")
	if err == nil {
		t.Error("ожидали ошибку, получили nil")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	mgr1 := NewJWTManager("secret-1", time.Hour)
	mgr2 := NewJWTManager("secret-2", time.Hour)

	token, _ := mgr1.BuildJWTString("user-1")

	_, err := mgr2.ParseToken(token)
	if err == nil {
		t.Error("ожидали ошибку для токена с чужим секретом")
	}
}

func TestParseToken_Expired(t *testing.T) {
	mgr := NewJWTManager("test-secret", -time.Hour) // уже истёк

	token, err := mgr.BuildJWTString("user-1")
	if err != nil {
		t.Fatalf("BuildJWTString: %v", err)
	}

	_, err = mgr.ParseToken(token)
	if err == nil {
		t.Error("ожидали ошибку для истёкшего токена")
	}
}
