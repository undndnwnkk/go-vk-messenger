package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

type UserRepositoryInterface interface {
	Create(ctx context.Context, user model.User) (string, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type UserService struct {
	repo       UserRepositoryInterface
	jwtService JwtService
}

func NewUserService(repo UserRepositoryInterface, jwtService JwtService) *UserService {
	return &UserService{repo: repo, jwtService: jwtService}
}

func (s *UserService) Register(ctx context.Context, req model.CreateUserRequest) (string, error) {
	// request validation
	if len(req.Username) < 1 || len(req.Username) > 100 {
		return "", ErrInvalidUsername
	}

	username := strings.ToLower(req.Username)
	existUser, err := s.repo.GetByUsername(ctx, username)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return "", fmt.Errorf("create user: %w", err)
	}

	if existUser != nil {
		return "", ErrUserAlreadyExists
	}

	if len(req.Password) < 8 {
		return "", ErrShortPassword
	}

	// password hashing
	pswdHash, err := hashPassword(req.Password)
	if err != nil {
		return "", fmt.Errorf("error while hashing: %w", err)
	}

	// creating user and sending to db
	user := model.User{Username: username, PasswordHash: pswdHash}
	id, err := s.repo.Create(ctx, user)
	if err != nil {
		return "", repoErrHandler(err)
	}

	token, err := s.jwtService.GenerateToken(id)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Login(ctx context.Context, req model.CreateUserRequest) (string, error) {
	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return "", repoErrHandler(err)
	}

	if !checkPasswordHash(req.Password, user.PasswordHash) {
		return "", ErrIncorrectPassword
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Me(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, repoErrHandler(err)
	}
	return user, nil
}

func hashPassword(password string) (string, error) {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	shaSum := hasher.Sum(nil)

	bytes, err := bcrypt.GenerateFromPassword(shaSum, bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	shaSum := hasher.Sum(nil)

	err := bcrypt.CompareHashAndPassword([]byte(hash), shaSum)
	return err == nil
}

func repoErrHandler(err error) error {
	if errors.Is(err, repository.ErrUsernameTaken) || errors.Is(err, repository.ErrUsernameNull) {
		return ErrUserAlreadyExists
	}
	if errors.Is(err, repository.ErrUserNotFound) {
		return ErrUserNotFound
	}
	return err
}

var (
	ErrInvalidUsername   = errors.New("invalid username")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrShortPassword     = errors.New("password too short")
	ErrIncorrectPassword = errors.New("username or password are incorrect")
)
