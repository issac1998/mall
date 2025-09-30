package queue

import (
	"context"
	"sync"
	"time"
)

// MemoryQueue memory-based queue implementation
type MemoryQueue struct {
	topics   map[string]*Topic
	config   *MemoryQueueConfig
	mu       sync.RWMutex
	closed   bool
	handlers map[string]MessageHandler
}

// Topic represents a message topic
type Topic struct {
	name     string
	messages chan []byte
	mu       sync.RWMutex
}

// MemoryQueueConfig memory queue configuration
type MemoryQueueConfig struct {
	BufferSize    int           `json:"buffer_size"`
	Topic         string        `json:"topic"`
	ProducerID    string        `json:"producer_id"`
	ConsumerGroup string        `json:"consumer_group"`
	Timeout       time.Duration `json:"timeout"`
}

// NewMemoryQueue creates a new memory queue instance
func NewMemoryQueue(config *MemoryQueueConfig) (*MemoryQueue, error) {
	if config == nil {
		config = &MemoryQueueConfig{
			BufferSize:    1000,
			Topic:         "seckill",
			ProducerID:    "seckill-producer",
			ConsumerGroup: "seckill-consumer",
			Timeout:       30 * time.Second,
		}
	}

	mq := &MemoryQueue{
		topics:   make(map[string]*Topic),
		config:   config,
		handlers: make(map[string]MessageHandler),
	}

	return mq, nil
}

// Publish publishes a message to the queue
func (mq *MemoryQueue) Publish(ctx context.Context, topic string, message []byte) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.closed {
		return ErrQueueClosed
	}

	// Get or create topic
	t, exists := mq.topics[topic]
	if !exists {
		t = &Topic{
			name:     topic,
			messages: make(chan []byte, mq.config.BufferSize),
		}
		mq.topics[topic] = t
	}

	// Send message with timeout
	select {
	case t.messages <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(mq.config.Timeout):
		return ErrPublishTimeout
	}
}

// Subscribe subscribes to messages from the queue
func (mq *MemoryQueue) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.closed {
		return ErrQueueClosed
	}

	// Get or create topic
	t, exists := mq.topics[topic]
	if !exists {
		t = &Topic{
			name:     topic,
			messages: make(chan []byte, mq.config.BufferSize),
		}
		mq.topics[topic] = t
	}

	// Store handler
	mq.handlers[topic] = handler

	// Start consuming messages in a goroutine
	go func() {
		for {
			select {
			case message := <-t.messages:
				if err := handler(ctx, topic, message); err != nil {
					// Log error but continue processing
					continue
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close closes the queue connections
func (mq *MemoryQueue) Close() error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if mq.closed {
		return nil
	}

	mq.closed = true

	// Close all topic channels
	for _, topic := range mq.topics {
		close(topic.messages)
	}

	// Clear topics and handlers
	mq.topics = make(map[string]*Topic)
	mq.handlers = make(map[string]MessageHandler)

	return nil
}

// Health checks the health of the queue
func (mq *MemoryQueue) Health() error {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	if mq.closed {
		return ErrQueueClosed
	}

	return nil
}

// GetStats returns queue statistics
func (mq *MemoryQueue) GetStats() *QueueStats {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	stats := &QueueStats{
		Topic:         mq.config.Topic,
		ProducerID:    mq.config.ProducerID,
		ConsumerGroup: mq.config.ConsumerGroup,
		Connected:     !mq.closed,
	}

	return stats
}