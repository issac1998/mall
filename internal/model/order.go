package model

import (
	"time"
)

// Order order model
type Order struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement;comment:订单ID" json:"id"`
	OrderNo        string     `gorm:"type:varchar(32);uniqueIndex;not null;comment:订单号" json:"order_no"`
	RequestID      string     `gorm:"type:varchar(32);uniqueIndex;not null;comment:请求ID（幂等）" json:"request_id"`
	UserID         uint64     `gorm:"type:bigint unsigned;not null;index;comment:用户ID" json:"user_id"`
	ActivityID     uint64     `gorm:"type:bigint unsigned;not null;index;comment:活动ID" json:"activity_id"`
	GoodsID        uint64     `gorm:"type:bigint unsigned;not null;comment:商品ID" json:"goods_id"`
	Quantity       int        `gorm:"type:int;not null;comment:购买数量" json:"quantity"`
	Price          int64      `gorm:"type:bigint;not null;comment:单价（分）" json:"price"`
	TotalAmount    int64      `gorm:"type:bigint;not null;comment:总金额（分）" json:"total_amount"`
	DiscountAmount int64      `gorm:"type:bigint;default:0;comment:优惠金额（分）" json:"discount_amount"`
	PaymentAmount  int64      `gorm:"type:bigint;not null;comment:实付金额（分）" json:"payment_amount"`
	Status         int8       `gorm:"type:tinyint;not null;default:1;index;comment:状态：1-待支付，2-已支付，3-已取消，4-已退款，5-已完成" json:"status"`
	PaymentMethod  *string    `gorm:"type:varchar(20);comment:支付方式" json:"payment_method,omitempty"`
	PaymentNo      *string    `gorm:"type:varchar(64);comment:支付流水号" json:"payment_no,omitempty"`
	PaidAt         *time.Time `gorm:"type:timestamp;comment:支付时间" json:"paid_at,omitempty"`
	ExpireAt       time.Time  `gorm:"type:timestamp;not null;index;comment:过期时间" json:"expire_at"`
	CancelReason   *string    `gorm:"type:varchar(255);comment:取消原因" json:"cancel_reason,omitempty"`
	Remark         *string    `gorm:"type:varchar(500);comment:备注" json:"remark,omitempty"`
	CreatedAt      time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;index;comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
	
	// 关联字段
	User           *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Activity       *SeckillActivity `gorm:"foreignKey:ActivityID" json:"activity,omitempty"`
	Goods          *Goods           `gorm:"foreignKey:GoodsID" json:"goods,omitempty"`
	Details        []OrderDetail    `gorm:"foreignKey:OrderID" json:"details,omitempty"`
}

// TableName set name
func (Order) TableName() string {
	return "orders"
}

// OrderDetail order detail model
type OrderDetail struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement;comment:详情ID" json:"id"`
	OrderID    uint64    `gorm:"type:bigint unsigned;not null;index;comment:订单ID" json:"order_id"`
	OrderNo    string    `gorm:"type:varchar(32);not null;index;comment:订单号" json:"order_no"`
	GoodsID    uint64    `gorm:"type:bigint unsigned;not null;index;comment:商品ID" json:"goods_id"`
	GoodsName  string    `gorm:"type:varchar(200);not null;comment:商品名称" json:"goods_name"`
	GoodsImage *string   `gorm:"type:varchar(255);comment:商品图片" json:"goods_image,omitempty"`
	Price      int64     `gorm:"type:bigint;not null;comment:单价（分）" json:"price"`
	Quantity   int       `gorm:"type:int;not null;comment:数量" json:"quantity"`
	Amount     int64     `gorm:"type:bigint;not null;comment:小计（分）" json:"amount"`
	CreatedAt  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
}

// TableName set name
func (OrderDetail) TableName() string {
	return "order_details"
}

// OrderStatus order status const
const (
	OrderStatusPending   = 1 // 待支付
	OrderStatusPaid      = 2 // 已支付
	OrderStatusCancelled = 3 // 已取消
	OrderStatusRefunded  = 4 // 已退款
	OrderStatusCompleted = 5 // 已完成
)


// PaymentMethod payment method const
const (
	PaymentMethodAlipay = "alipay"
	PaymentMethodWechat = "wechat"
	PaymentMethodBank   = "bank"
	PaymentMethodBalance = "balance"
)

// IsPending check order is pending
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsPaid check order is paid
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid
}

// IsCancelled check order is cancelled
func (o *Order) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}

// IsRefunded check order is refunded		
func (o *Order) IsRefunded() bool {
	return o.Status == OrderStatusRefunded
}

// IsCompleted check order is completed
func (o *Order) IsCompleted() bool {
	return o.Status == OrderStatusCompleted
}

// IsExpired check order is expired
func (o *Order) IsExpired() bool {
	return time.Now().After(o.ExpireAt)
}

// CanPay check order can pay
func (o *Order) CanPay() bool {
	return o.IsPending() && !o.IsExpired()
}

// CanCancel check order can cancel
func (o *Order) CanCancel() bool {
	return o.IsPending() || o.IsPaid()
}

// CanRefund check order can refund
func (o *Order) CanRefund() bool {
	return o.IsPaid()
}

// GetTotalAmountYuan get total amount in yuan
func (o *Order) GetTotalAmountYuan() float64 {
	return float64(o.TotalAmount) / 100
}

// GetPaymentAmountYuan get payment amount in yuan
func (o *Order) GetPaymentAmountYuan() float64 {
	return float64(o.PaymentAmount) / 100
}

// GetDiscountAmountYuan get discount amount in yuan
func (o *Order) GetDiscountAmountYuan() float64 {
	return float64(o.DiscountAmount) / 100
}

// GetPriceYuan get price in yuan
func (o *Order) GetPriceYuan() float64 {
	return float64(o.Price) / 100
}

// GetPriceYuan get price in yuan
func (od *OrderDetail) GetPriceYuan() float64 {
	return float64(od.Price) / 100
}

// GetAmountYuan get amount in yuan
func (od *OrderDetail) GetAmountYuan() float64 {
	return float64(od.Amount) / 100
}