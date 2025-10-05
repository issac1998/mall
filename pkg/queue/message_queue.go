package queue

import (
	"context"
	"sync"
)

// MessageQueue message queue interface
type MessageQueue interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, message []byte) error
	// Consume consumes a message from a topic
	Consume(ctx context.Context, topic string) ([]byte, error)
	// Close closes the queue
	Close() error
}

// MemoryMessageQueue in-memory message queue implementation
type MemoryMessageQueue struct {
	queues map[string]chan []byte
	mu     sync.RWMutex
	closed bool
}

// NewMemoryMessageQueue creates a new in-memory message queue
func NewMemoryMessageQueue() *MemoryMessageQueue {
	return &MemoryMessageQueue{
		queues: make(map[string]chan []byte),
	}
}

// Publish publishes a message to a topic
func (q *MemoryMessageQueue) Publish(ctx context.Context, topic string, message []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	// Get or create queue for topic
	queue, ok := q.queues[topic]
	if !ok {
		queue = make(chan []byte, 1000) // Buffer size 1000
		q.queues[topic] = queue
	}

	// Try to send message (non-blocking)
	select {
	case queue <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue full, send in background
		go func() {
			select {
			case queue <- message:
			case <-ctx.Done():
			}
		}()
		return nil
	}
}

// Consume consumes a message from a topic
func (q *MemoryMessageQueue) Consume(ctx context.Context, topic string) ([]byte, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return nil, ErrQueueClosed
	}

	queue, ok := q.queues[topic]
	if !ok {
		// Create queue if doesn't exist
		q.mu.RUnlock()
		q.mu.Lock()
		queue = make(chan []byte, 1000)
		q.queues[topic] = queue
		q.mu.Unlock()
		q.mu.RLock()
	}
	q.mu.RUnlock()

	// Wait for message
	select {
	case message := <-queue:
		return message, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the queue
func (q *MemoryMessageQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	for _, queue := range q.queues {
		close(queue)
	}
	return nil
}

