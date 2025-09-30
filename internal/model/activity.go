package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// SeckillActivity model
type SeckillActivity struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement;comment:活动ID" json:"id"`
	Name             string    `gorm:"type:varchar(200);not null;comment:活动名称" json:"name"`
	GoodsID          uint64    `gorm:"type:bigint unsigned;not null;index;comment:商品ID" json:"goods_id"`
	Price            int64     `gorm:"type:bigint;not null;comment:秒杀价格（分）" json:"price"`
	Stock            int       `gorm:"type:int;not null;comment:秒杀库存" json:"stock"`
	Sold             int       `gorm:"type:int;not null;default:0;comment:已售数量" json:"sold"`
	StartTime        time.Time `gorm:"type:timestamp;not null;index;comment:开始时间" json:"start_time"`
	EndTime          time.Time `gorm:"type:timestamp;not null;index;comment:结束时间" json:"end_time"`
	LimitPerUser     int       `gorm:"type:int;not null;default:1;comment:每人限购数量" json:"limit_per_user"`
	Status           int8      `gorm:"type:tinyint;not null;default:0;index;comment:状态：0-未开始，1-进行中，2-已结束，3-已暂停，4-已取消" json:"status"`
	
	PrewarmTime      *time.Time `gorm:"type:timestamp;comment:预热时间" json:"prewarm_time,omitempty"`
	PrewarmStatus    int8       `gorm:"type:tinyint;default:0;comment:预热状态：0-未预热，1-已预热" json:"prewarm_status"`
	Priority         int        `gorm:"type:int;default:0;index;comment:优先级（影响资源分配）" json:"priority"`
	RiskLevel        int8       `gorm:"type:tinyint;default:1;comment:风险级别：1-5" json:"risk_level"`
	MaxQPS           int        `gorm:"type:int;default:10000;comment:最大QPS限制" json:"max_qps"`
	MaxConcurrent    int        `gorm:"type:int;default:5000;comment:最大并发数" json:"max_concurrent"`
	ShardCount       int        `gorm:"type:int;default:1;comment:库存分片数" json:"shard_count"`
	ShardStrategy    string     `gorm:"type:varchar(20);default:'hash';comment:分片策略" json:"shard_strategy"`
	DegradeThreshold float64    `gorm:"type:decimal(5,4);default:0.5;comment:降级阈值" json:"degrade_threshold"`
	DegradeStrategy  string     `gorm:"type:varchar(50);default:'queue';comment:降级策略" json:"degrade_strategy"`
	GrayStrategy     *string    `gorm:"type:varchar(50);comment:灰度策略" json:"gray_strategy,omitempty"`
	GrayRatio        float64    `gorm:"type:decimal(5,4);default:0;comment:灰度比例" json:"gray_ratio"`
	GrayWhitelist    JSONObject `gorm:"type:json;comment:灰度白名单" json:"gray_whitelist,omitempty"`
	ExtConfig        JSONObject `gorm:"type:json;comment:扩展配置" json:"ext_config,omitempty"`
	
	CreatedAt        time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
	
	Goods            *Goods     `gorm:"foreignKey:GoodsID" json:"goods,omitempty"`
}

// TableName set name	
func (SeckillActivity) TableName() string {
	return "seckill_activities"
}

// ActivityStatus activity status const
const (
	ActivityStatusNotStarted = 0 // 未开始
	ActivityStatusRunning    = 1 // 进行中
	ActivityStatusEnded      = 2 // 已结束
	ActivityStatusPaused     = 3 // 已暂停
	ActivityStatusCancelled  = 4 // 已取消
)

// PrewarmStatus prewarm status const
const (
	PrewarmStatusNot    = 0 // 未预热
	PrewarmStatusDone   = 1 // 已预热
)

// JSONObject custom json object type
type JSONObject map[string]interface{}

// Value implement driver.Valuer interface
func (j JSONObject) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implement sql.Scanner interface
func (j *JSONObject) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONObject", value)
	}
	
	return json.Unmarshal(bytes, j)
}

// IsNotStarted check if activity is not started	
func (a *SeckillActivity) IsNotStarted() bool {
	return a.Status == ActivityStatusNotStarted
}

// IsRunning check if activity is running	
func (a *SeckillActivity) IsRunning() bool {
	return a.Status == ActivityStatusRunning
}

// IsEnded check if activity is ended	
func (a *SeckillActivity) IsEnded() bool {
	return a.Status == ActivityStatusEnded
}

// IsPaused check if activity is paused	
func (a *SeckillActivity) IsPaused() bool {
	return a.Status == ActivityStatusPaused
}

// IsCancelled check if activity is cancelled	
func (a *SeckillActivity) IsCancelled() bool {
	return a.Status == ActivityStatusCancelled
}

// HasStock check if activity has stock	
func (a *SeckillActivity) HasStock() bool {
	return a.Stock > a.Sold
}

// GetRemainingStock get remaining stock
func (a *SeckillActivity) GetRemainingStock() int {
	remaining := a.Stock - a.Sold
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetSellRate get sell rate
func (a *SeckillActivity) GetSellRate() float64 {
	if a.Stock == 0 {
		return 0
	}
	return float64(a.Sold) / float64(a.Stock) * 100
}

// GetPriceYuan get price in yuan
func (a *SeckillActivity) GetPriceYuan() float64 {
	return float64(a.Price) / 100
}

// IsPrewarmed check if activity is prewarmed	
func (a *SeckillActivity) IsPrewarmed() bool {
	return a.PrewarmStatus == PrewarmStatusDone
}

// ShouldPrewarm check if activity should prewarm	
func (a *SeckillActivity) ShouldPrewarm() bool {
	if a.PrewarmTime == nil {
		return false
	}
	return time.Now().After(*a.PrewarmTime) && !a.IsPrewarmed()
}

// ShouldStart check if activity should start	
func (a *SeckillActivity) ShouldStart() bool {
	return time.Now().After(a.StartTime) && a.IsNotStarted()
}

// ShouldEnd check if activity should end	
func (a *SeckillActivity) ShouldEnd() bool {
	return time.Now().After(a.EndTime) && a.IsRunning()
}