package message

import (
	"context"
	"database/sql"
	"time"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/google/uuid"
)

type (
	CreateMessageParams = proto.CreateMessageParams
	Message             = proto.Message
	Attachment          = proto.Attachment
	ToolCall            = proto.ToolCall
	ToolResult          = proto.ToolResult
	ContentPart         = proto.ContentPart
	TextContent         = proto.TextContent
	BinaryContent       = proto.BinaryContent
	FinishReason        = proto.FinishReason
	Finish              = proto.Finish
)

const (
	Assistant = proto.Assistant
	User      = proto.User
	System    = proto.System
	Tool      = proto.Tool

	FinishReasonEndTurn          = proto.FinishReasonEndTurn
	FinishReasonMaxTokens        = proto.FinishReasonMaxTokens
	FinishReasonToolUse          = proto.FinishReasonToolUse
	FinishReasonCanceled         = proto.FinishReasonCanceled
	FinishReasonError            = proto.FinishReasonError
	FinishReasonPermissionDenied = proto.FinishReasonPermissionDenied

	FinishReasonUnknown = proto.FinishReasonUnknown
)

type Service interface {
	pubsub.Suscriber[Message]
	Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
	Update(ctx context.Context, message Message) error
	Get(ctx context.Context, id string) (Message, error)
	List(ctx context.Context, sessionID string) ([]Message, error)
	Delete(ctx context.Context, id string) error
	DeleteSessionMessages(ctx context.Context, sessionID string) error
}

type service struct {
	*pubsub.Broker[Message]
	q db.Querier
}

func NewService(q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[Message](),
		q:      q,
	}
}

func (s *service) Delete(ctx context.Context, id string) error {
	message, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	err = s.q.DeleteMessage(ctx, message.ID)
	if err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, message)
	return nil
}

func (s *service) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	if params.Role != proto.Assistant {
		params.Parts = append(params.Parts, proto.Finish{
			Reason: "stop",
		})
	}
	partsJSON, err := proto.MarshallParts(params.Parts)
	if err != nil {
		return Message{}, err
	}
	dbMessage, err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      string(params.Role),
		Parts:     string(partsJSON),
		Model:     sql.NullString{String: string(params.Model), Valid: true},
		Provider:  sql.NullString{String: params.Provider, Valid: params.Provider != ""},
	})
	if err != nil {
		return Message{}, err
	}
	message, err := s.fromDBItem(dbMessage)
	if err != nil {
		return Message{}, err
	}
	s.Publish(pubsub.CreatedEvent, message)
	return message, nil
}

func (s *service) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	messages, err := s.List(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if message.SessionID == sessionID {
			err = s.Delete(ctx, message.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) Update(ctx context.Context, message Message) error {
	parts, err := proto.MarshallParts(message.Parts)
	if err != nil {
		return err
	}
	finishedAt := sql.NullInt64{}
	if f := message.FinishPart(); f != nil {
		finishedAt.Int64 = f.Time
		finishedAt.Valid = true
	}
	err = s.q.UpdateMessage(ctx, db.UpdateMessageParams{
		ID:         message.ID,
		Parts:      string(parts),
		FinishedAt: finishedAt,
	})
	if err != nil {
		return err
	}
	message.UpdatedAt = time.Now().Unix()
	s.Publish(pubsub.UpdatedEvent, message)
	return nil
}

func (s *service) Get(ctx context.Context, id string) (Message, error) {
	dbMessage, err := s.q.GetMessage(ctx, id)
	if err != nil {
		return Message{}, err
	}
	return s.fromDBItem(dbMessage)
}

func (s *service) List(ctx context.Context, sessionID string) ([]Message, error) {
	dbMessages, err := s.q.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func (s *service) fromDBItem(item db.Message) (Message, error) {
	parts, err := proto.UnmarshallParts([]byte(item.Parts))
	if err != nil {
		return Message{}, err
	}
	return Message{
		ID:        item.ID,
		SessionID: item.SessionID,
		Role:      proto.MessageRole(item.Role),
		Parts:     parts,
		Model:     item.Model.String,
		Provider:  item.Provider.String,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, nil
}
