package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("NewMemoryQueue", func(t *testing.T) {
		// Test with default config
		mq, err := NewMemoryQueue(nil)
		assert.NoError(t, err)
		assert.NotNil(t, mq)
		defer mq.Close()

		// Test with custom config
		config := &MemoryQueueConfig{
			BufferSize:    500,
			Topic:         "test-topic",
			ProducerID:    "test-producer",
			ConsumerGroup: "test-consumer",
			Timeout:       10 * time.Second,
		}
		mq2, err := NewMemoryQueue(config)
		assert.NoError(t, err)
		assert.NotNil(t, mq2)
		defer mq2.Close()
	})

	t.Run("PublishAndSubscribe", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)
		defer mq.Close()

		topic := "test-topic"
		message := []byte("test message")
		received := make(chan []byte, 1)

		// Subscribe first
		handler := func(ctx context.Context, topic string, msg []byte) error {
			received <- msg
			return nil
		}

		err = mq.Subscribe(ctx, topic, handler)
		assert.NoError(t, err)

		// Give subscriber time to start
		time.Sleep(10 * time.Millisecond)

		// Publish message
		err = mq.Publish(ctx, topic, message)
		assert.NoError(t, err)

		// Wait for message
		select {
		case receivedMsg := <-received:
			assert.Equal(t, message, receivedMsg)
		case <-time.After(time.Second):
			t.Fatal("Message not received within timeout")
		}
	})

	t.Run("MultipleMessages", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)
		defer mq.Close()

		topic := "multi-topic"
		messageCount := 10
		received := make(chan []byte, messageCount)

		// Subscribe
		handler := func(ctx context.Context, topic string, msg []byte) error {
			received <- msg
			return nil
		}

		err = mq.Subscribe(ctx, topic, handler)
		assert.NoError(t, err)

		// Give subscriber time to start
		time.Sleep(10 * time.Millisecond)

		// Publish multiple messages
		for i := 0; i < messageCount; i++ {
			message := []byte("message " + string(rune(i+'0')))
			err = mq.Publish(ctx, topic, message)
			assert.NoError(t, err)
		}

		// Receive all messages
		receivedCount := 0
		timeout := time.After(2 * time.Second)
		for receivedCount < messageCount {
			select {
			case <-received:
				receivedCount++
			case <-timeout:
				t.Fatalf("Only received %d out of %d messages", receivedCount, messageCount)
			}
		}
	})

	t.Run("MultipleTopic", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)
		defer mq.Close()

		topic1 := "topic1"
		topic2 := "topic2"
		message1 := []byte("message for topic1")
		message2 := []byte("message for topic2")

		received1 := make(chan []byte, 1)
		received2 := make(chan []byte, 1)

		// Subscribe to both topics
		handler1 := func(ctx context.Context, topic string, msg []byte) error {
			received1 <- msg
			return nil
		}
		handler2 := func(ctx context.Context, topic string, msg []byte) error {
			received2 <- msg
			return nil
		}

		err = mq.Subscribe(ctx, topic1, handler1)
		assert.NoError(t, err)
		err = mq.Subscribe(ctx, topic2, handler2)
		assert.NoError(t, err)

		// Give subscribers time to start
		time.Sleep(10 * time.Millisecond)

		// Publish to both topics
		err = mq.Publish(ctx, topic1, message1)
		assert.NoError(t, err)
		err = mq.Publish(ctx, topic2, message2)
		assert.NoError(t, err)

		// Verify messages received on correct topics
		select {
		case receivedMsg := <-received1:
			assert.Equal(t, message1, receivedMsg)
		case <-time.After(time.Second):
			t.Fatal("Message not received on topic1")
		}

		select {
		case receivedMsg := <-received2:
			assert.Equal(t, message2, receivedMsg)
		case <-time.After(time.Second):
			t.Fatal("Message not received on topic2")
		}
	})

	t.Run("PublishTimeout", func(t *testing.T) {
		config := &MemoryQueueConfig{
			BufferSize: 1, // Small buffer to trigger timeout
			Timeout:    10 * time.Millisecond,
		}
		mq, err := NewMemoryQueue(config)
		require.NoError(t, err)
		defer mq.Close()

		topic := "timeout-topic"

		// Fill the buffer
		err = mq.Publish(ctx, topic, []byte("message1"))
		assert.NoError(t, err)

		// This should timeout
		err = mq.Publish(ctx, topic, []byte("message2"))
		assert.Error(t, err)
		assert.Equal(t, ErrPublishTimeout, err)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)
		defer mq.Close()

		topic := "cancel-topic"
		cancelCtx, cancel := context.WithCancel(ctx)

		// Subscribe with cancellable context
		var wg sync.WaitGroup
		wg.Add(1)
		handler := func(ctx context.Context, topic string, msg []byte) error {
			defer wg.Done()
			return nil
		}

		err = mq.Subscribe(cancelCtx, topic, handler)
		assert.NoError(t, err)

		// Give subscriber time to start
		time.Sleep(10 * time.Millisecond)

		// Publish message
		err = mq.Publish(ctx, topic, []byte("test"))
		assert.NoError(t, err)

		// Wait for message to be processed
		wg.Wait()

		// Cancel context
		cancel()

		// Give time for goroutine to exit
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("Close", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)

		topic := "close-topic"

		// Subscribe
		handler := func(ctx context.Context, topic string, msg []byte) error {
			return nil
		}
		err = mq.Subscribe(ctx, topic, handler)
		assert.NoError(t, err)

		// Close queue
		err = mq.Close()
		assert.NoError(t, err)

		// Operations should fail after close
		err = mq.Publish(ctx, topic, []byte("test"))
		assert.Equal(t, ErrQueueClosed, err)

		err = mq.Subscribe(ctx, topic, handler)
		assert.Equal(t, ErrQueueClosed, err)

		// Close again should not error
		err = mq.Close()
		assert.NoError(t, err)
	})

	t.Run("Health", func(t *testing.T) {
		mq, err := NewMemoryQueue(nil)
		require.NoError(t, err)

		// Should be healthy initially
		err = mq.Health()
		assert.NoError(t, err)

		// Should be unhealthy after close
		mq.Close()
		err = mq.Health()
		assert.Equal(t, ErrQueueClosed, err)
	})

	t.Run("GetStats", func(t *testing.T) {
		config := &MemoryQueueConfig{
			Topic:         "stats-topic",
			ProducerID:    "stats-producer",
			ConsumerGroup: "stats-consumer",
		}
		mq, err := NewMemoryQueue(config)
		require.NoError(t, err)
		defer mq.Close()

		stats := mq.GetStats()
		assert.Equal(t, "stats-topic", stats.Topic)
		assert.Equal(t, "stats-producer", stats.ProducerID)
		assert.Equal(t, "stats-consumer", stats.ConsumerGroup)
		assert.True(t, stats.Connected)

		// After close
		mq.Close()
		stats = mq.GetStats()
		assert.False(t, stats.Connected)
	})
}

func TestMemoryQueueInterface(t *testing.T) {
	t.Run("ImplementsQueueInterface", func(t *testing.T) {
		var _ Queue = (*MemoryQueue)(nil)
	})
}