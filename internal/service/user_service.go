package service

import (
	"context"
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
	jwtService *JWTService
}

func NewUserService(repo UserRepositoryInterface, jwtService *JWTService) *UserService {
	return &UserService{repo: repo, jwtService: jwtService}
}

func (s *UserService) Register(ctx context.Context, req model.CreateUserRequest) (string, error) {
	// request validation
	if len(req.Username) < 1 || len(req.Username) > 100 {
		return "", ErrInvalidUsername
	}

	username := strings.ToLower(req.Username)
	if len([]byte(req.Password)) < 8 {
		return "", ErrShortPassword
	}
	if len([]byte(req.Password)) > 72 {
		return "", ErrLongPassword
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
	user, err := s.repo.GetByUsername(ctx, strings.ToLower(req.Username))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by username: %w", err)
	}

	if !checkPasswordHash(req.Password, user.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Me(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, repoErrHandler(err)
	}
	if user == nil {
		return nil, ErrNilUser
	}
	return user, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func repoErrHandler(err error) error {
	if errors.Is(err, repository.ErrUsernameTaken) {
		return ErrUserAlreadyExists
	}
	if errors.Is(err, repository.ErrUserNotFound) {
		return ErrUserNotFound
	}
	return err
}

var (
	ErrInvalidUsername    = errors.New("invalid username")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrShortPassword      = errors.New("password too short")
	ErrLongPassword       = errors.New("password too long")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNilUser            = errors.New("repository returned nil user")
)
