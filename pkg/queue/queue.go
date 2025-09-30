package queue

import (
	"context"
	"errors"
)

// Queue defines the interface for message queue operations
type Queue interface {
	// Publish publishes a message to the specified topic
	Publish(ctx context.Context, topic string, message []byte) error
	
	// Subscribe subscribes to messages from the specified topic
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	
	// Close closes the queue connections
	Close() error
	
	// Health checks the health of the queue
	Health() error
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx context.Context, topic string, message []byte) error

// TransactionHandler handles transactional messages
type TransactionHandler interface {
	ProcessMessage(ctx context.Context, topic string, message []byte) error
	CommitTransaction(ctx context.Context, topic string, message []byte) error
	RollbackTransaction(ctx context.Context, topic string, message []byte) error
}

// QueueStats represents queue statistics
type QueueStats struct {
	Topic         string `json:"topic"`
	ProducerID    string `json:"producer_id"`
	ConsumerGroup string `json:"consumer_group"`
	Connected     bool   `json:"connected"`
	MessagesSent  int64  `json:"messages_sent"`
	MessagesRecv  int64  `json:"messages_received"`
}

// Common errors
var (
	ErrQueueClosed              = errors.New("queue is closed")
	ErrProducerNotInitialized   = errors.New("producer not initialized")
	ErrConsumerNotInitialized   = errors.New("consumer not initialized")
	ErrInvalidConfiguration     = errors.New("invalid configuration")
	ErrConnectionFailed         = errors.New("connection failed")
	ErrPublishTimeout          = errors.New("publish timeout")
	ErrSubscribeTimeout        = errors.New("subscribe timeout")
)