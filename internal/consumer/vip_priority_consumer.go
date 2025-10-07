package consumer

import (
	"context"
	"time"

	"seckill/internal/service/order"
	"seckill/pkg/log"
	"seckill/pkg/queue"
)

// VIPPriorityConsumer VIP priority order consumer
// Consumes from VIP queue first, then normal queue
type VIPPriorityConsumer struct {
	orderService order.OrderService
	messageQueue queue.MessageQueue
	stopCh       chan struct{}
	vipWorkers   int
	normalWorkers int
}

// NewVIPPriorityConsumer creates a VIP priority consumer
func NewVIPPriorityConsumer(
	orderService order.OrderService,
	messageQueue queue.MessageQueue,
	vipWorkers int,
	normalWorkers int,
) *VIPPriorityConsumer {
	if vipWorkers <= 0 {
		vipWorkers = 3
	}
	if normalWorkers <= 0 {
		normalWorkers = 10
	}
	
	return &VIPPriorityConsumer{
		orderService:  orderService,
		messageQueue:  messageQueue,
		stopCh:        make(chan struct{}),
		vipWorkers:    vipWorkers,
		normalWorkers: normalWorkers,
	}
}

// Start starts the VIP priority consumer
func (c *VIPPriorityConsumer) Start(ctx context.Context) {
	log.WithFields(map[string]interface{}{
		"vip_workers":    c.vipWorkers,
		"normal_workers": c.normalWorkers,
	}).Info("Starting VIP priority order consumer")

	// Start VIP workers (higher priority, dedicated workers)
	for i := 0; i < c.vipWorkers; i++ {
		go c.consumeVIP(ctx, i)
	}

	// Start normal workers (consume from both queues, VIP first)
	for i := 0; i < c.normalWorkers; i++ {
		go c.consumeWithPriority(ctx, i)
	}
}

// consumeVIP consumes only from VIP queue
func (c *VIPPriorityConsumer) consumeVIP(ctx context.Context, workerID int) {
	log.WithFields(map[string]interface{}{
		"worker_id": workerID,
		"type":      "vip",
	}).Info("VIP worker started")

	for {
		select {
		case <-c.stopCh:
			log.WithFields(map[string]interface{}{
				"worker_id": workerID,
			}).Info("VIP worker stopped")
			return
		case <-ctx.Done():
			log.WithFields(map[string]interface{}{
				"worker_id": workerID,
			}).Info("VIP worker context cancelled")
			return
		default:
			c.processMessage(ctx, "seckill_orders_vip", workerID, "VIP")
		}
	}
}

// consumeWithPriority consumes with priority: VIP first, then normal
func (c *VIPPriorityConsumer) consumeWithPriority(ctx context.Context, workerID int) {
	log.WithFields(map[string]interface{}{
		"worker_id": workerID,
		"type":      "priority",
	}).Info("Priority worker started")

	for {
		select {
		case <-c.stopCh:
			log.WithFields(map[string]interface{}{
				"worker_id": workerID,
			}).Info("Priority worker stopped")
			return
		case <-ctx.Done():
			log.WithFields(map[string]interface{}{
				"worker_id": workerID,
			}).Info("Priority worker context cancelled")
			return
		default:
			// Try VIP queue first (with short timeout)
			vipProcessed := c.tryProcessMessage(ctx, "seckill_orders_vip", workerID, "VIP", 100*time.Millisecond)
			
			if !vipProcessed {
				// If no VIP message, try normal queue
				c.processMessage(ctx, "seckill_orders", workerID, "Normal")
			}
		}
	}
}

// tryProcessMessage tries to process a message with timeout
func (c *VIPPriorityConsumer) tryProcessMessage(ctx context.Context, topic string, workerID int, queueType string, timeout time.Duration) bool {
	consumeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	messageData, err := c.messageQueue.Consume(consumeCtx, topic)
	if err != nil {
		// Timeout or error, no message available
		return false
	}

	// Process message
	if err := c.orderService.ConsumeOrderMessage(ctx, messageData); err != nil {
		log.WithFields(map[string]interface{}{
			"worker_id": workerID,
			"queue":     queueType,
			"error":     err.Error(),
		}).Error("Failed to process message")
	} else {
		log.WithFields(map[string]interface{}{
			"worker_id": workerID,
			"queue":     queueType,
		}).Debug("Message processed successfully")
	}

	return true
}

// processMessage processes a message from queue
func (c *VIPPriorityConsumer) processMessage(ctx context.Context, topic string, workerID int, queueType string) {
	consumeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	messageData, err := c.messageQueue.Consume(consumeCtx, topic)
	if err != nil {
		if err == context.DeadlineExceeded {
			// Timeout is normal when queue is empty
			return
		}
		log.WithFields(map[string]interface{}{
			"worker_id": workerID,
			"queue":     queueType,
			"error":     err.Error(),
		}).Error("Failed to consume message")
		time.Sleep(1 * time.Second)
		return
	}

	// Process message
	if err := c.orderService.ConsumeOrderMessage(ctx, messageData); err != nil {
		log.WithFields(map[string]interface{}{
			"worker_id": workerID,
			"queue":     queueType,
			"error":     err.Error(),
		}).Error("Failed to process message")
	} else {
		log.WithFields(map[string]interface{}{
			"worker_id": workerID,
			"queue":     queueType,
		}).Debug("Message processed successfully")
	}
}

// Stop stops the consumer
func (c *VIPPriorityConsumer) Stop() {
	close(c.stopCh)
	log.Info("VIP priority consumer stopped")
}

