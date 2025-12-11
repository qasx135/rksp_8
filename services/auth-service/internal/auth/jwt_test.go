package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	secret := []byte("testsecret")
	tokenStr, err := GenerateToken(1, "user@example.com", secret, time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("token not valid: %v", err)
	}

	if claims["email"] != "user@example.com" {
		t.Fatalf("unexpected email claim: %v", claims["email"])
	}
}
