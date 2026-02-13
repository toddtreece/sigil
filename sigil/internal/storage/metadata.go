package storage

import (
	"context"
	"errors"
	"time"
)

var ErrBlockAlreadyExists = errors.New("block metadata already exists")

type BlockMetadataStore interface {
	InsertBlock(ctx context.Context, meta BlockMeta) error
	ListBlocks(ctx context.Context, tenantID string, from, to time.Time) ([]BlockMeta, error)
}

type ConversationStore interface {
	ListConversations(ctx context.Context, tenantID string) ([]Conversation, error)
	GetConversation(ctx context.Context, tenantID, conversationID string) (*Conversation, error)
}
