package service

import (
	"context"
	"errors"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
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

func (s *ChatService) CreateDirectChat(ctx context.Context, userID1, userID2 string, chatType model.ChatType) (*model.Chat, error) {
	if userID1 == userID2 {
		return nil, ErrCommonUser
	}

	if _, err := s.userService.GetUserByID(ctx, userID2); err != nil {
		return nil, ErrUserNotFound
	}

	chat, err := s.repo.GetOrCreateDirect(ctx, userID1, userID2)
	if err != nil {
		// TODO handle error
		return nil, err
	}

	return chat, nil
}

func (s *ChatService) CreateGroupChat(ctx context.Context, creatorID string, req model.CreateGroupChat) (*model.Chat, error) {
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}
	chat, err := s.repo.CreateGroup(ctx, creatorID, req.Title, req.UserIDs)
	if err != nil {
		// TODO handle error
		return nil, err
	}

	return chat, nil
}

func (s *ChatService) GetChats(ctx context.Context, userID string) ([]model.Chat, error) {
	chats, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		// TODO handle error
		return nil, err
	}

	return chats, err
}

func (s *ChatService) GetChatByID(ctx context.Context, userID, chatID string) (*model.Chat, error) {
	chat, err := s.repo.GetByIDForUser(ctx, chatID, userID)
	if err != nil {
		// TODO handle error
		return nil, err
	}

	return chat, nil
}

func (s *ChatService) GetMembersByChatID(ctx context.Context, chatID string) ([]model.ChatMember, error) {
	members, err := s.repo.ListMembers(ctx, chatID)
	if err != nil {
		// TODO handle error
		return nil, err
	}

	return members, nil
}

func (s *ChatService) AddMember(ctx context.Context, creatorID, chatID, userID string, role model.ChatRole) error {
	creator, err := s.repo.GetMember(ctx, chatID, creatorID)
	if err != nil {
		// TODO handle error
		return err
	}
	if creator.Role != "admin" {
		return ErrNotAdmin
	}

	if err = s.repo.AddMember(ctx, chatID, userID, role); err != nil {
		// TODO handle error
		return err
	}
	return nil
}

func (s *ChatService) RemoveMember(ctx context.Context, creatorID, chatID, userID string) error {
	creator, err := s.repo.GetMember(ctx, chatID, creatorID)
	if err != nil {
		// TODO handle error
		return err
	}
	if creator.Role != "admin" {
		return ErrNotAdmin
	}

	if err = s.repo.RemoveMember(ctx, chatID, userID); err != nil {
		// TODO handle error
		return err
	}
	return nil
}

var (
	ErrCommonUser   = errors.New("chat must be with different users")
	ErrInvalidTitle = errors.New("title must be not empty")
	ErrNotAdmin     = errors.New("you need to be admin to add new users into chat")
)
