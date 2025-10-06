package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"seckill/internal/model"
)

func main() {
	// Database connection
	dsn := "root:@tcp(localhost:3306)/seckill_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	ctx := context.Background()

	// Test GetByID
	fmt.Println("=== Testing GetByID ===")
	var activity1 model.SeckillActivity
	err = db.WithContext(ctx).Where("id = ?", 1).First(&activity1).Error
	if err != nil {
		log.Fatal("Failed to find activity:", err)
	}
	fmt.Printf("GetByID - Activity 1 limit_per_user: %d\n", activity1.LimitPerUser)
	
	// Test GetByIDWithGoods
	fmt.Println("\n=== Testing GetByIDWithGoods ===")
	var activity2 model.SeckillActivity
	err = db.WithContext(ctx).
		Preload("Goods").
		Where("id = ?", 1).
		First(&activity2).Error
	if err != nil {
		log.Fatal("Failed to find activity with goods:", err)
	}
	fmt.Printf("GetByIDWithGoods - Activity 1 limit_per_user: %d\n", activity2.LimitPerUser)

	// Test JSON marshaling
	fmt.Println("\n=== Testing JSON Marshaling ===")
	jsonData, err := json.Marshal(activity2)
	if err != nil {
		log.Fatal("Failed to marshal JSON:", err)
	}
	fmt.Printf("JSON: %s\n", string(jsonData))

	// Test JSON unmarshaling
	fmt.Println("\n=== Testing JSON Unmarshaling ===")
	var activity3 model.SeckillActivity
	err = json.Unmarshal(jsonData, &activity3)
	if err != nil {
		log.Fatal("Failed to unmarshal JSON:", err)
	}
	fmt.Printf("After unmarshal - Activity 1 limit_per_user: %d\n", activity3.LimitPerUser)
}