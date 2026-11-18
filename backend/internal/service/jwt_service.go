package service

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type JWTService struct {
	Secret    []byte
	accessTTL time.Duration
}

type Claims struct {
	UserID string `json:"-"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, ttl time.Duration) *JWTService {
	return &JWTService{Secret: []byte(secret), accessTTL: ttl}
}

func (s *JWTService) GenerateToken(userID string) (string, error) {
	now := time.Now()
	exp := now.Add(s.accessTTL)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.Secret)
	if err != nil {
		return "", ErrJwt
	}

	return tokenString, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrUnknownSigningMethod
		}
		return s.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, ErrJwt
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.UserID == "" {
			claims.UserID = claims.Subject
		}
		if claims.UserID == "" {
			return nil, ErrInvalidToken
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *JWTService) GetUserID(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

var (
	ErrJwt                  = errors.New("problem with jwt")
	ErrUnknownSigningMethod = errors.New("unknown jwt signing method")
	ErrInvalidToken         = errors.New("invalid token")
)
