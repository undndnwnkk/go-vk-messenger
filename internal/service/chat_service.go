package service

import (
	"context"
	"errors"
	"strings"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

type ChatRepositoryInterface interface {
	CreateGroup(
		ctx context.Context,
		creatorID string,
		title string,
		memberIDs []string,
	) (*model.Chat, error)

	GetOrCreateDirect(
		ctx context.Context,
		userID1 string,
		userID2 string,
	) (*model.Chat, error)

	GetByID(
		ctx context.Context,
		chatID string,
	) (*model.Chat, error)

	GetByIDForUser(
		ctx context.Context,
		chatID string,
		userID string,
	) (*model.Chat, error)

	ListByUser(
		ctx context.Context,
		userID string,
	) ([]model.Chat, error)

	GetMember(
		ctx context.Context,
		chatID string,
		userID string,
	) (*model.ChatMember, error)

	ListMembers(
		ctx context.Context,
		chatID string,
	) ([]model.ChatMember, error)

	AddMember(
		ctx context.Context,
		chatID string,
		userID string,
		role model.ChatRole,
	) error

	RemoveMember(
		ctx context.Context,
		chatID string,
		userID string,
	) error
}

type ChatService struct {
	userService UserService
	repo        ChatRepositoryInterface
}

func NewChatService(repo ChatRepositoryInterface, userService UserService) *ChatService {
	return &ChatService{repo: repo, userService: userService}
}

func (s *ChatService) CreateDirectChat(ctx context.Context, userID1, userID2 string) (*model.Chat, error) {
	if userID1 == userID2 {
		return nil, ErrCommonUser
	}

	if _, err := s.userService.GetUserByID(ctx, userID2); err != nil {
		return nil, err
	}

	chat, err := s.repo.GetOrCreateDirect(ctx, userID1, userID2)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) CreateGroupChat(ctx context.Context, creatorID string, req model.CreateGroupChat) (*model.Chat, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}
	chat, err := s.repo.CreateGroup(ctx, creatorID, req.Title, req.UserIDs)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) GetChats(ctx context.Context, userID string) ([]model.Chat, error) {
	chats, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chats, err
}

func (s *ChatService) GetChatByID(ctx context.Context, userID, chatID string) (*model.Chat, error) {
	chat, err := s.repo.GetByIDForUser(ctx, chatID, userID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) GetMembersByChatID(ctx context.Context, currentUserID, chatID string) ([]model.ChatMember, error) {
	if _, err := s.repo.GetByIDForUser(ctx, chatID, currentUserID); err != nil {
		return nil, chatRepoError(err)
	}
	members, err := s.repo.ListMembers(ctx, chatID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return members, nil
}

func (s *ChatService) requireGroupAdmin(ctx context.Context, currentUserID, chatID string) error {
	chat, err := s.repo.GetByID(ctx, chatID)
	if err != nil {
		return chatRepoError(err)
	}
	if chat.Type != model.Group {
		return ErrCannotModifyDirect
	}
	member, err := s.repo.GetMember(ctx, chatID, currentUserID)
	if errors.Is(err, repository.ErrMemberNotFound) {
		return ErrNotAdmin
	}
	if err != nil {
		return chatRepoError(err)
	}
	if member.Role != model.Admin {
		return ErrNotAdmin
	}
	return nil
}

func (s *ChatService) AddMember(ctx context.Context, currentUserID, chatID, userID string) error {
	if err := s.requireGroupAdmin(ctx, currentUserID, chatID); err != nil {
		return err
	}
	if _, err := s.userService.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return chatRepoError(s.repo.AddMember(ctx, chatID, userID, model.Member))
}

func (s *ChatService) RemoveMember(ctx context.Context, currentUserID, chatID, userID string) error {
	if err := s.requireGroupAdmin(ctx, currentUserID, chatID); err != nil {
		return err
	}
	return chatRepoError(s.repo.RemoveMember(ctx, chatID, userID))
}

func chatRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrChatNotFound):
		return ErrChatNotFound
	case errors.Is(err, repository.ErrMemberNotFound):
		return ErrMemberNotFound
	case errors.Is(err, repository.ErrMemberAlreadyExists):
		return ErrAlreadyMember
	default:
		return err
	}
}

var (
	ErrChatNotFound       = errors.New("chat not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrAlreadyMember      = errors.New("member already exists")
	ErrCannotModifyDirect = errors.New("cannot modify direct chat members")
	ErrCommonUser         = errors.New("chat must be with different users")
	ErrInvalidTitle       = errors.New("title must be not empty")
	ErrNotAdmin           = errors.New("you need to be admin to add new users into chat")
)
