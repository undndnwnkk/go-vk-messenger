package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
)

const messageUser = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const messageChat = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

type messageRepoStub struct {
	reply                        *model.MessageReply
	message                      *model.Message
	mutationCalled               bool
	mutationErr                  error
	messageID                    int64
	called                       bool
	createCount                  int
	chat, sender, content, query string
	before                       *int64
	limit                        int
	messages                     []model.Message
	err                          error
}

func (r *messageRepoStub) GetByID(_ context.Context, chat string, id int64) (*model.Message, error) {
	r.called, r.chat, r.messageID = true, chat, id
	if r.message != nil {
		return r.message, r.err
	}
	return &model.Message{ID: id, ChatID: chat, SenderID: messageUser}, r.err
}

func (r *messageRepoStub) Edit(_ context.Context, chat, sender string, id int64, content string) (*model.Message, error) {
	r.mutationCalled, r.chat, r.sender, r.messageID, r.content = true, chat, sender, id, content
	now := time.Now()
	return &model.Message{ID: id, ChatID: chat, SenderID: sender, Content: content, Edited: true, EditedAt: &now}, r.mutationErr
}

func (r *messageRepoStub) Delete(_ context.Context, chat, sender string, id int64) (*model.Message, error) {
	r.mutationCalled, r.chat, r.sender, r.messageID = true, chat, sender, id
	now := time.Now()
	return &model.Message{ID: id, ChatID: chat, SenderID: sender, Deleted: true, DeletedAt: &now}, r.mutationErr
}

func TestMessageMutations(t *testing.T) {
	failure := errors.New("database failure")
	for _, operation := range []string{"edit", "delete"} {
		for _, tc := range []struct {
			name                                  string
			id                                    int64
			message                               *model.Message
			accessErr, repoErr, mutationErr, want error
			read, mutate                          bool
		}{
			{name: "Author", id: 42, read: true, mutate: true},
			{name: "InvalidID", id: 0, want: ErrInvalidMessageID},
			{name: "NegativeID", id: -1, want: ErrInvalidMessageID},
			{name: "OutsiderOrRemovedAuthor", id: 42, accessErr: repository.ErrChatNotFound, want: ErrChatNotFound},
			{name: "AccessFailure", id: 42, accessErr: failure, want: failure},
			{name: "MissingOrOtherChat", id: 42, repoErr: repository.ErrMessageNotFound, want: ErrMessageNotFound, read: true},
			{name: "NotAuthor", id: 42, message: &model.Message{SenderID: "other"}, want: ErrNotMessageAuthor, read: true},
			{name: "Deleted", id: 42, message: &model.Message{SenderID: messageUser, Deleted: true}, want: ErrMessageNotFound, read: true},
			{name: "ReadFailure", id: 42, repoErr: failure, want: failure, read: true},
			{name: "MutationFailure", id: 42, mutationErr: failure, want: failure, read: true, mutate: true},
			{name: "ConcurrentDelete", id: 42, mutationErr: repository.ErrMessageNotFound, want: ErrMessageNotFound, read: true, mutate: true},
		} {
			t.Run(operation+tc.name, func(t *testing.T) {
				repo := &messageRepoStub{message: tc.message, err: tc.repoErr, mutationErr: tc.mutationErr}
				svc := messageTestService(repo, tc.accessErr)
				var msg *model.Message
				var err error
				if operation == "edit" {
					msg, err = svc.Edit(context.Background(), strings.ToUpper(messageUser), strings.ToUpper(messageChat), tc.id, model.EditMessageRequest{Content: " updated "})
				} else {
					msg, err = svc.Delete(context.Background(), strings.ToUpper(messageUser), strings.ToUpper(messageChat), tc.id)
				}
				if !errors.Is(err, tc.want) || repo.called != tc.read || repo.mutationCalled != tc.mutate {
					t.Fatalf("err=%v read=%v mutate=%v", err, repo.called, repo.mutationCalled)
				}
				if tc.want == nil {
					if msg.ID != 42 || repo.chat != messageChat || repo.sender != messageUser || repo.messageID != 42 {
						t.Fatalf("message=%+v repo=%+v", msg, repo)
					}
					if operation == "edit" && (msg.Content != " updated " || !msg.Edited || msg.EditedAt == nil) {
						t.Fatalf("edited message=%+v", msg)
					}
					if operation == "delete" && (msg.Content != "" || !msg.Deleted || msg.DeletedAt == nil) {
						t.Fatalf("deleted message=%+v", msg)
					}
				}
			})
		}
	}
}

func TestMessageEditContentValidation(t *testing.T) {
	for _, tc := range []struct {
		content string
		want    error
	}{
		{"", ErrEmptyMessage}, {" \n\t", ErrEmptyMessage},
		{strings.Repeat("я", 4001), ErrMessageTooLong},
		{strings.Repeat("😀", 4000), nil},
	} {
		repo := &messageRepoStub{}
		_, err := messageTestService(repo, nil).Edit(context.Background(), messageUser, messageChat, 1, model.EditMessageRequest{Content: tc.content})
		if !errors.Is(err, tc.want) || repo.mutationCalled != (tc.want == nil) || repo.called != (tc.want == nil) {
			t.Fatalf("err=%v called=%v mutated=%v", err, repo.called, repo.mutationCalled)
		}
	}
}

func (r *messageRepoStub) Create(_ context.Context, chat, sender, content string, replyToMessageID *int64) (*model.Message, error) {
	r.called, r.chat, r.sender, r.content = true, chat, sender, content
	r.createCount++
	return &model.Message{ID: 1, ChatID: chat, SenderID: sender, Content: content, ReplyToMessageID: replyToMessageID, ReplyTo: r.reply}, r.err
}
func (r *messageRepoStub) ListBefore(_ context.Context, chat string, before *int64, limit int) ([]model.Message, error) {
	r.called, r.chat, r.before, r.limit = true, chat, before, limit
	return r.messages, r.err
}
func (r *messageRepoStub) Search(_ context.Context, chat, query string, limit int) ([]model.Message, error) {
	r.called, r.chat, r.query, r.limit = true, chat, query, limit
	return r.messages, r.err
}
func messageTestService(repo *messageRepoStub, accessErr error) *MessageService {
	return NewMessageService(repo, *NewChatService(&chatRepoStub{accessErr: accessErr}, UserService{}))
}
func TestMessageSend(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          error
	}{
		{"Empty", "", ErrEmptyMessage}, {"Whitespace", " \t\n", ErrEmptyMessage},
		{"TooLong", strings.Repeat("a", 4001), ErrMessageTooLong},
		{"UnicodeTooLong", strings.Repeat("\U0001F600", 4001), ErrMessageTooLong},
		{"UnicodeBoundary", strings.Repeat("\u044f\U0001F600", 2000), nil}, {"Success", " hello ", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &messageRepoStub{}
			msg, err := messageTestService(repo, nil).Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: tc.content})
			if !errors.Is(err, tc.want) || repo.called != (tc.want == nil) {
				t.Fatalf("error=%v called=%v", err, repo.called)
			}
			if tc.want == nil && (msg.ID != 1 || msg.Content != tc.content || repo.chat != messageChat || repo.sender != messageUser || repo.content != tc.content) {
				t.Fatalf("message=%+v repo=%+v", msg, repo)
			}
		})
	}
}
func TestMessageHistory(t *testing.T) {
	cursor := int64(8)
	zero := int64(0)
	negative := int64(-1)
	for _, tc := range []struct {
		name      string
		limit     int
		before    *int64
		messages  []model.Message
		want      error
		wantLimit int
		wantIDs   []int64
		next      *int64
	}{
		{name: "Empty", limit: 50, messages: []model.Message{}, wantLimit: 51, wantIDs: []int64{}},
		{name: "ZeroLimit", limit: 0, want: ErrInvalidLimit},
		{name: "NegativeLimit", limit: -1, want: ErrInvalidLimit},
		{name: "ZeroCursor", limit: 3, before: &zero, want: ErrInvalidCursor},
		{name: "NegativeCursor", limit: 3, before: &negative, want: ErrInvalidCursor},
		{name: "Cursor", limit: 3, messages: []model.Message{{ID: 10}, {ID: 9}, {ID: 8}, {ID: 7}}, wantLimit: 4, wantIDs: []int64{10, 9, 8}, next: &cursor},
		{name: "SubsequentPage", limit: 3, before: &cursor, messages: []model.Message{{ID: 7}, {ID: 6}}, wantLimit: 4, wantIDs: []int64{7, 6}},
		{name: "ExactFinalPage", limit: 3, messages: []model.Message{{ID: 3}, {ID: 2}, {ID: 1}}, wantLimit: 4, wantIDs: []int64{3, 2, 1}},
		{name: "SmallLimit", limit: 10, messages: []model.Message{}, wantLimit: 11, wantIDs: []int64{}},
		{name: "ClampedLimit", limit: 101, messages: []model.Message{}, wantLimit: 101, wantIDs: []int64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &messageRepoStub{messages: tc.messages}
			page, err := messageTestService(repo, nil).History(context.Background(), messageUser, messageChat, tc.before, tc.limit)
			if !errors.Is(err, tc.want) || repo.called != (tc.want == nil) {
				t.Fatalf("error=%v called=%v", err, repo.called)
			}
			if tc.want != nil {
				return
			}
			ids := make([]int64, 0, len(page.Messages))
			for _, msg := range page.Messages {
				ids = append(ids, msg.ID)
			}
			if !reflect.DeepEqual(ids, tc.wantIDs) || !reflect.DeepEqual(page.NextCursor, tc.next) || repo.limit != tc.wantLimit || repo.chat != messageChat || !reflect.DeepEqual(repo.before, tc.before) {
				t.Fatalf("page=%+v repo=%+v", page, repo)
			}
			if len(tc.messages) == 0 && page.Messages == nil {
				t.Fatal("nil empty page")
			}
		})
	}
}
func TestMessageSearch(t *testing.T) {
	for _, query := range []string{"", " \t\n", " hello "} {
		t.Run(query, func(t *testing.T) {
			repo := &messageRepoStub{messages: []model.Message{{ID: 5, Content: "hello"}}}
			msgs, err := messageTestService(repo, nil).Search(context.Background(), messageUser, messageChat, query)
			if strings.TrimSpace(query) == "" {
				if !errors.Is(err, ErrEmptySearchQuery) || repo.called {
					t.Fatalf("error=%v called=%v", err, repo.called)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(msgs, repo.messages) || repo.query != "hello" || repo.chat != messageChat || repo.limit != 50 {
				t.Fatalf("error=%v repo=%+v", err, repo)
			}
		})
	}
}
func TestMessageAccessAndErrorPropagation(t *testing.T) {
	failure := errors.New("database failure")
	for _, operation := range []string{"send", "history", "search"} {
		for _, tc := range []struct {
			name, chat               string
			accessErr, repoErr, want error
			called                   bool
		}{
			{"Outsider", messageChat, repository.ErrChatNotFound, nil, ErrChatNotFound, false},
			{"InvalidUUID", "invalid", nil, nil, ErrInvalidID, false},
			{"AccessFailure", messageChat, failure, nil, failure, false},
			{"RepositoryFailure", messageChat, nil, failure, failure, true},
		} {
			t.Run(operation+tc.name, func(t *testing.T) {
				repo := &messageRepoStub{err: tc.repoErr}
				svc := messageTestService(repo, tc.accessErr)
				ctx := context.Background()
				var err error
				switch operation {
				case "send":
					_, err = svc.Send(ctx, messageUser, tc.chat, model.CreateMessageRequest{Content: "hello"})
				case "history":
					_, err = svc.History(ctx, messageUser, tc.chat, nil, 50)
				case "search":
					_, err = svc.Search(ctx, messageUser, tc.chat, "hello")
				}
				if !errors.Is(err, tc.want) || repo.called != tc.called {
					t.Fatalf("error=%v called=%v", err, repo.called)
				}
			})
		}
	}
}

func TestMessageSendRateLimit(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	limiter := NewMessageRateLimiterWithClock(func() time.Time { return now })
	repo := &messageRepoStub{}
	svc := NewMessageServiceWithLimiter(repo, *NewChatService(&chatRepoStub{}, UserService{}), limiter)

	for i := 0; i < MessageRateLimitCount; i++ {
		if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("11th send error = %v, want rate limited", err)
	}
	if repo.createCount != MessageRateLimitCount {
		t.Fatalf("repository creates = %d, want %d", repo.createCount, MessageRateLimitCount)
	}

	now = now.Add(MessageRateLimitWindow + time.Millisecond)
	if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); err != nil {
		t.Fatalf("send after window: %v", err)
	}
	if repo.createCount != MessageRateLimitCount+1 {
		t.Fatalf("repository creates after window = %d", repo.createCount)
	}
}

func TestMessageSendRateLimitIsPerUser(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	limiter := NewMessageRateLimiterWithClock(func() time.Time { return now })
	repo := &messageRepoStub{}
	svc := NewMessageServiceWithLimiter(repo, *NewChatService(&chatRepoStub{}, UserService{}), limiter)

	for i := 0; i < MessageRateLimitCount; i++ {
		if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); err != nil {
			t.Fatalf("alice send %d: %v", i, err)
		}
	}
	if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("alice after limit error = %v, want rate limited", err)
	}

	const bob = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	if _, err := svc.Send(context.Background(), bob, messageChat, model.CreateMessageRequest{Content: "hello"}); err != nil {
		t.Fatalf("bob should not inherit alice limit: %v", err)
	}
}

func TestMessageRateLimiterConcurrentAllow(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	limiter := NewMessageRateLimiterWithClock(func() time.Time { return now })
	const attempts = 100

	var allowed int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if limiter.Allow(messageUser) {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if allowed != MessageRateLimitCount {
		t.Fatalf("allowed concurrent attempts = %d, want %d", allowed, MessageRateLimitCount)
	}
}

func TestMessageSendRateLimitCountsFailedAccessAttempts(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	limiter := NewMessageRateLimiterWithClock(func() time.Time { return now })
	repo := &messageRepoStub{}
	svc := NewMessageServiceWithLimiter(repo, *NewChatService(&chatRepoStub{accessErr: repository.ErrChatNotFound}, UserService{}), limiter)

	for i := 0; i < MessageRateLimitCount; i++ {
		if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); !errors.Is(err, ErrChatNotFound) {
			t.Fatalf("failed access attempt %d error = %v, want chat not found", i, err)
		}
	}
	if _, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"}); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("after failed access attempts error = %v, want rate limited", err)
	}
	if repo.called {
		t.Fatal("failed access attempts reached message repository")
	}
}

func TestMessageHistoryClampedPageSize(t *testing.T) {
	messages := make([]model.Message, 101)
	for i := range messages {
		messages[i].ID = int64(101 - i)
	}
	repo := &messageRepoStub{messages: messages}
	page, err := messageTestService(repo, nil).History(context.Background(), messageUser, messageChat, nil, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if repo.limit != 101 || len(page.Messages) != 100 || page.NextCursor == nil || *page.NextCursor != 2 {
		t.Fatalf("page size=%d cursor=%v query limit=%d", len(page.Messages), page.NextCursor, repo.limit)
	}
}
