package consumer

import (
	"context"
	"time"

	"seckill/internal/service/order"
	"seckill/pkg/log"
	"seckill/pkg/queue"
)

// OrderConsumer order message consumer
type OrderConsumer struct {
	orderService order.OrderService
	messageQueue queue.MessageQueue
	stopCh       chan struct{}
}

// NewOrderConsumer creates an order consumer
func NewOrderConsumer(orderService order.OrderService, messageQueue queue.MessageQueue) *OrderConsumer {
	return &OrderConsumer{
		orderService: orderService,
		messageQueue: messageQueue,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the consumer
func (c *OrderConsumer) Start(ctx context.Context) {
	log.Info("Starting order consumer")
	
	go func() {
		for {
			select {
			case <-c.stopCh:
				log.Info("Order consumer stopped")
				return
			case <-ctx.Done():
				log.Info("Order consumer context cancelled")
				return
			default:
				// Consume message with timeout
				consumeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				messageData, err := c.messageQueue.Consume(consumeCtx, "seckill_orders")
				cancel()
				
				if err != nil {
					if err == context.DeadlineExceeded {
						// Timeout is normal, continue
						continue
					}
					log.WithFields(map[string]interface{}{
						"error": err.Error(),
					}).Error("Failed to consume order message")
					time.Sleep(1 * time.Second)
					continue
				}

				// Process message
				if err := c.orderService.ConsumeOrderMessage(ctx, messageData); err != nil {
					log.WithFields(map[string]interface{}{
						"error": err.Error(),
					}).Error("Failed to process order message")
				}
			}
		}
	}()
}

// Stop stops the consumer
func (c *OrderConsumer) Stop() {
	close(c.stopCh)
}