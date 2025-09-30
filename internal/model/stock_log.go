package model

import (
	"time"
)

// StockLog stock log model
type StockLog struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement;comment:日志ID" json:"id"`
	ActivityID    uint64    `gorm:"type:bigint unsigned;not null;index;comment:活动ID" json:"activity_id"`
	GoodsID       uint64    `gorm:"type:bigint unsigned;not null;index;comment:商品ID" json:"goods_id"`
	OperationType int8      `gorm:"type:tinyint;not null;comment:操作类型：1-扣减，2-回补，3-同步" json:"operation_type"`
	Quantity      int       `gorm:"type:int;not null;comment:数量（正数为增加，负数为减少）" json:"quantity"`
	BeforeStock   int       `gorm:"type:int;not null;comment:操作前库存" json:"before_stock"`
	AfterStock    int       `gorm:"type:int;not null;comment:操作后库存" json:"after_stock"`
	RequestID     *string   `gorm:"type:varchar(32);index;comment:请求ID" json:"request_id,omitempty"`
	OrderNo       *string   `gorm:"type:varchar(32);index;comment:订单号" json:"order_no,omitempty"`
	Operator      *string   `gorm:"type:varchar(50);comment:操作人" json:"operator,omitempty"`
	Remark        *string   `gorm:"type:varchar(255);comment:备注" json:"remark,omitempty"`
	CreatedAt     time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;index;comment:创建时间" json:"created_at"`
	

	Activity      *SeckillActivity `gorm:"foreignKey:ActivityID" json:"activity,omitempty"`
	Goods         *Goods           `gorm:"foreignKey:GoodsID" json:"goods,omitempty"`
}

// TableName set name	
func (StockLog) TableName() string {
	return "stock_logs"
}

// OperationType operation type const
const (
	OperationTypeDeduct = 1 // 扣减
	OperationTypeRevert = 2 // 回补
	OperationTypeSync   = 3 // 同步
)

// IsDeduct check if operation is deduct	
func (sl *StockLog) IsDeduct() bool {
	return sl.OperationType == OperationTypeDeduct
}

// IsRevert check if operation is revert	
func (sl *StockLog) IsRevert() bool {
	return sl.OperationType == OperationTypeRevert
}

// IsSync check if operation is sync	
func (sl *StockLog) IsSync() bool {
	return sl.OperationType == OperationTypeSync
}

// GetOperationTypeName get operation type name	
func (sl *StockLog) GetOperationTypeName() string {
	switch sl.OperationType {
	case OperationTypeDeduct:
		return "扣减"
	case OperationTypeRevert:
		return "回补"
	case OperationTypeSync:
		return "同步"
	default:
		return "未知"
	}
}