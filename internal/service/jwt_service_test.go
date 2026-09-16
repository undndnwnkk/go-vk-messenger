package service

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTContainsRegisteredClaims(t *testing.T) {
	svc := JwtService{
		Secret:    []byte("test-secret"),
		accessTTL: 15,
	}

	tokenString, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return svc.Secret, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token is invalid")
	}
	if claims.Subject != "user-1" {
		t.Fatalf("sub = %q, want %q", claims.Subject, "user-1")
	}
	if claims.IssuedAt == nil {
		t.Fatal("iat claim is missing")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("exp claim is missing")
	}
}

func TestJWTRejectsExpiredAndTamperedToken(t *testing.T) {
	svc := JwtService{
		Secret:    []byte("test-secret"),
		accessTTL: -1,
	}

	expiredToken, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, err := svc.ValidateToken(expiredToken); !errors.Is(err, ErrJwt) {
		t.Fatalf("expired token error = %v, want %v", err, ErrJwt)
	}

	validSvc := JwtService{
		Secret:    []byte("test-secret"),
		accessTTL: 15,
	}
	validToken, err := validSvc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, err := validSvc.ValidateToken(validToken + "tampered"); !errors.Is(err, ErrJwt) {
		t.Fatalf("tampered token error = %v, want %v", err, ErrJwt)
	}
}
