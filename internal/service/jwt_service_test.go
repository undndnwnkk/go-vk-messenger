package service

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTContainsRegisteredClaims(t *testing.T) {
	svc := NewJWTService("test-secret", 15*time.Minute)

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
	svc := NewJWTService("test-secret", -time.Minute)

	expiredToken, err := svc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, err := svc.ValidateToken(expiredToken); !errors.Is(err, ErrJwt) {
		t.Fatalf("expired token error = %v, want %v", err, ErrJwt)
	}

	validSvc := NewJWTService("test-secret", 15*time.Minute)
	validToken, err := validSvc.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, err := validSvc.ValidateToken(validToken + "tampered"); !errors.Is(err, ErrJwt) {
		t.Fatalf("tampered token error = %v, want %v", err, ErrJwt)
	}
}

func TestJWTRejectsUnexpectedSigningMethod(t *testing.T) {
	svc := NewJWTService("test-secret", 15*time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	})
	tokenString, err := token.SignedString(svc.Secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ValidateToken(tokenString); !errors.Is(err, ErrJwt) {
		t.Fatalf("unexpected signing method error = %v, want %v", err, ErrJwt)
	}
}
