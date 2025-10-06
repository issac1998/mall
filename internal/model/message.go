package model

// OrderMessage order message for MQ
type OrderMessage struct {
	RequestID  string  `json:"request_id"`  // Request ID (idempotency)
	DeductID   string  `json:"deduct_id"`   // Deduct ID (for TCC)
	UserID     uint64  `json:"user_id"`     // User ID
	ActivityID uint64  `json:"activity_id"` // Activity ID
	GoodsID    uint64  `json:"goods_id"`    // Goods ID
	Quantity   int     `json:"quantity"`    // Quantity
	Price      float64 `json:"price"`       // Unit price
	IsVIP      bool    `json:"is_vip"`      // Is VIP user
	IP         string  `json:"ip"`          // User IP
	DeviceID   string  `json:"device_id"`   // Device ID
	Timestamp  int64   `json:"timestamp"`   // Timestamp
	TraceID    string  `json:"trace_id"`    // Trace ID
}

// StockMessage stock message for MQ
type StockMessage struct {
	ActivityID uint64 `json:"activity_id"` // Activity ID
	GoodsID    uint64 `json:"goods_id"`    // Goods ID
	Operation  string `json:"operation"`   // Operation type: deduct/revert/sync
	Quantity   int    `json:"quantity"`    // Quantity
	RequestID  string `json:"request_id"`  // Request ID
	Timestamp  int64  `json:"timestamp"`   // Timestamp
}

// NotificationMessage notification message for MQ
type NotificationMessage struct {
	UserID    uint64            `json:"user_id"`    // User ID
	Type      string            `json:"type"`       // Notification type
	Title     string            `json:"title"`      // Title
	Content   string            `json:"content"`    // Content
	Data      map[string]interface{} `json:"data"` // Extension data
	Channels  []string          `json:"channels"`   // Notification channels: sms/email/push
	Timestamp int64             `json:"timestamp"`  // Timestamp
}

