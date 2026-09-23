package service

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

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

	MarkRead(
		ctx context.Context,
		chatID string,
		userID string,
		messageID int64,
	) (*model.ChatReadState, bool, error)
}

type ChatService struct {
	userService UserService
	repo        ChatRepositoryInterface
}

func NewChatService(repo ChatRepositoryInterface, userService UserService) *ChatService {
	return &ChatService{repo: repo, userService: userService}
}

func (s *ChatService) CreateDirectChat(ctx context.Context, userID1, userID2 string) (*model.Chat, error) {
	if err := normalizeChatIDs(&userID1, &userID2); err != nil {
		return nil, err
	}
	if userID1 == userID2 {
		return nil, ErrCommonUser
	}

	if err := s.requireTargetUser(ctx, userID2); err != nil {
		return nil, err
	}

	chat, err := s.repo.GetOrCreateDirect(ctx, userID1, userID2)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) CreateGroupChat(ctx context.Context, creatorID string, req model.CreateGroupChat) (*model.Chat, error) {
	if err := normalizeChatIDs(&creatorID); err != nil {
		return nil, err
	}
	req.Title = strings.TrimSpace(req.Title)
	if !utf8.ValidString(req.Title) || utf8.RuneCountInString(req.Title) < 1 || utf8.RuneCountInString(req.Title) > 100 {
		return nil, ErrInvalidTitle
	}
	memberIDs := make([]string, 0, len(req.UserIDs))
	seen := map[string]bool{creatorID: true}
	for _, id := range req.UserIDs {
		if err := normalizeChatIDs(&id); err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		memberIDs = append(memberIDs, id)
	}
	for _, id := range memberIDs {
		if err := s.requireTargetUser(ctx, id); err != nil {
			return nil, err
		}
	}
	chat, err := s.repo.CreateGroup(ctx, creatorID, req.Title, memberIDs)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) GetChats(ctx context.Context, userID string) ([]model.Chat, error) {
	if err := normalizeChatIDs(&userID); err != nil {
		return nil, err
	}
	chats, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chats, err
}

func (s *ChatService) GetChatByID(ctx context.Context, userID, chatID string) (*model.Chat, error) {
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, err
	}
	chat, err := s.repo.GetByIDForUser(ctx, chatID, userID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return chat, nil
}

func (s *ChatService) GetMembersByChatID(ctx context.Context, currentUserID, chatID string) ([]model.ChatMember, error) {
	if err := normalizeChatIDs(&currentUserID, &chatID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByIDForUser(ctx, chatID, currentUserID); err != nil {
		return nil, chatRepoError(err)
	}
	members, err := s.repo.ListMembers(ctx, chatID)
	if err != nil {
		return nil, chatRepoError(err)
	}

	return members, nil
}

func (s *ChatService) MarkRead(ctx context.Context, userID, chatID string, messageID int64) (*model.ChatReadState, bool, error) {
	if messageID <= 0 {
		return nil, false, ErrInvalidMessageID
	}
	if err := normalizeChatIDs(&userID, &chatID); err != nil {
		return nil, false, err
	}
	state, advanced, err := s.repo.MarkRead(ctx, chatID, userID, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return nil, false, ErrChatNotFound
		}
		return nil, false, chatRepoError(err)
	}
	return state, advanced, nil
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
	if err := normalizeChatIDs(&currentUserID, &chatID, &userID); err != nil {
		return err
	}
	if err := s.requireGroupAdmin(ctx, currentUserID, chatID); err != nil {
		return err
	}
	if err := s.requireTargetUser(ctx, userID); err != nil {
		return err
	}
	return chatRepoError(s.repo.AddMember(ctx, chatID, userID, model.Member))
}

func (s *ChatService) RemoveMember(ctx context.Context, currentUserID, chatID, userID string) error {
	if err := normalizeChatIDs(&currentUserID, &chatID, &userID); err != nil {
		return err
	}
	if err := s.requireGroupAdmin(ctx, currentUserID, chatID); err != nil {
		return err
	}
	if currentUserID == userID {
		return ErrCannotRemoveSelf
	}
	return chatRepoError(s.repo.RemoveMember(ctx, chatID, userID))
}

// Accept the canonical UUID shape (case insensitive) and normalize before comparisons.
func normalizeChatIDs(ids ...*string) error {
	for _, id := range ids {
		value := *id
		if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
			return ErrInvalidID
		}
		compact := value[:8] + value[9:13] + value[14:18] + value[19:23] + value[24:]
		if _, err := hex.DecodeString(compact); err != nil {
			return ErrInvalidID
		}
		*id = strings.ToLower(value)
	}
	return nil
}

func (s *ChatService) requireTargetUser(ctx context.Context, id string) error {
	_, err := s.userService.GetUserByID(ctx, id)
	if errors.Is(err, ErrUserNotFound) {
		return ErrTargetUserNotFound
	}
	return err
}

func chatRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrChatNotFound):
		return ErrChatNotFound
	case errors.Is(err, repository.ErrMemberNotFound):
		return ErrMemberNotFound
	case errors.Is(err, repository.ErrMemberAlreadyExists):
		return ErrAlreadyMember
	case errors.Is(err, repository.ErrMessageNotFound):
		return ErrMessageNotFound
	default:
		return err
	}
}

var (
	ErrInvalidID          = errors.New("invalid UUID")
	ErrTargetUserNotFound = errors.New("target user not found")
	ErrCannotRemoveSelf   = errors.New("cannot remove yourself from the group")
	ErrChatNotFound       = errors.New("chat not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrMessageNotFound    = errors.New("message not found")
	ErrInvalidMessageID   = errors.New("invalid message id")
	ErrAlreadyMember      = errors.New("member already exists")
	ErrCannotModifyDirect = errors.New("cannot modify direct chat members")
	ErrCommonUser         = errors.New("chat must be with different users")
	ErrInvalidTitle       = errors.New("title must contain 1 to 100 characters after trimming")
	ErrNotAdmin           = errors.New("you need to be admin to add new users into chat")
)
