package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Goods goods model
type Goods struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;comment:商品ID" json:"id"`
	Name        string    `gorm:"type:varchar(200);not null;comment:商品名称" json:"name"`
	Description *string   `gorm:"type:text;comment:商品描述" json:"description,omitempty"`
	Category    *string   `gorm:"type:varchar(50);index;comment:商品分类" json:"category,omitempty"`
	Brand       *string   `gorm:"type:varchar(50);index;comment:品牌" json:"brand,omitempty"`
	Images      JSONArray `gorm:"type:json;comment:商品图片（JSON数组）" json:"images,omitempty"`
	Price       int64     `gorm:"type:bigint;not null;comment:原价（分）" json:"price"`
	Stock       int       `gorm:"type:int;not null;default:0;comment:库存数量" json:"stock"`
	Sales       int       `gorm:"type:int;not null;default:0;comment:销量" json:"sales"`
	Status      int8      `gorm:"type:tinyint;not null;default:1;index;comment:状态：1-上架，2-下架，3-删除" json:"status"`
	CreatedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;index;comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName set name
func (Goods) TableName() string {
	return "goods"
}

// GoodsStatus goods status const		
const (
	GoodsStatusOnSale   = 1 // 上架
	GoodsStatusOffSale  = 2 // 下架
	GoodsStatusDeleted  = 3 // 删除
)

// JSONArray custom json array type
type JSONArray []string

// Value implement driver.Valuer interface
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implement sql.Scanner interface
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONArray", value)
	}
	
	return json.Unmarshal(bytes, j)
}

// IsOnSale check if goods is on sale
func (g *Goods) IsOnSale() bool {
	return g.Status == GoodsStatusOnSale
}

// IsOffSale check if goods is off sale
func (g *Goods) IsOffSale() bool {
	return g.Status == GoodsStatusOffSale
}

// IsDeleted check if goods is deleted
func (g *Goods) IsDeleted() bool {
	return g.Status == GoodsStatusDeleted
}

// HasStock check if goods has stock
func (g *Goods) HasStock() bool {
	return g.Stock > 0
}

// GetPriceYuan get price in yuan
func (g *Goods) GetPriceYuan() float64 {
	return float64(g.Price) / 100
}