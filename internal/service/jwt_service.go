package service

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"strconv"
	"time"
)

type JwtService struct {
	Secret    []byte
	accessTTL int
}

type Claims struct {
	UserID string
	jwt.RegisteredClaims
}

func NewJwtService() *JwtService {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		key = "supersecretforjwtnothanks"
	}
	ttl := os.Getenv("JWT_TTL")
	var ttlInt int
	if ttl == "" {
		ttlInt = 15
	} else {
		ttlInt, _ = strconv.Atoi(ttl)
	}
	return &JwtService{Secret: []byte(key), accessTTL: ttlInt}
}

func (s *JwtService) GenerateToken(userID string) (string, error) {
	exp := time.Now().Add(time.Duration(s.accessTTL) * time.Minute)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.Secret)
	if err != nil {
		return "", ErrJwt
	}

	return tokenString, nil
}

func (s *JwtService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnknownSigningMethod
		}
		return s.Secret, nil
	})

	if err != nil {
		return nil, ErrJwt
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *JwtService) GetUserID(tokenString string) (string, error) {
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
