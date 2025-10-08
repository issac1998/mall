package seckill

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBloomFilterDebug(t *testing.T) {
	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	defer redisClient.Close()

	// Clear test database
	redisClient.FlushDB(context.Background())

	// Create inventory manager
	inventory, err := NewMultiLevelInventory(redisClient)
	require.NoError(t, err)

	ctx := context.Background()
	activityID := uint64(12345)

	fmt.Printf("=== Debug Bloom Filter Operations ===\n")

	// Test 1: Check initial state
	exists := inventory.LocalCheck(ctx, activityID)
	fmt.Printf("1. Initial LocalCheck(%d): %v (should be false)\n", activityID, exists)
	assert.False(t, exists, "Activity should not exist initially")

	// Test 2: Add to bloom filter
	inventory.AddToBloomFilter(activityID)
	require.NoError(t, err)
	fmt.Printf("2. Added %d to bloom filter\n", activityID)

	// Test 3: Check after adding
	exists = inventory.LocalCheck(ctx, activityID)
	fmt.Printf("3. LocalCheck(%d) after adding: %v (should be true)\n", activityID, exists)
	assert.True(t, exists, "Activity should exist after adding to bloom filter")

	// Test 4: Mark as sold out
	err = inventory.MarkSoldOut(activityID)
	require.NoError(t, err)
	fmt.Printf("4. Marked %d as sold out\n", activityID)

	// Test 5: Check after marking sold out
	exists = inventory.LocalCheck(ctx, activityID)
	fmt.Printf("5. LocalCheck(%d) after sold out: %v (should be false)\n", activityID, exists)
	assert.False(t, exists, "Activity should return false after being marked sold out")

	// Test 6: Test direct bloom filter operations
	bloomKey := fmt.Sprintf("goods:{%d}", activityID)
	fmt.Printf("6. Testing direct bloom filter with key: %s\n", bloomKey)

	// Add directly to bloom filter
	inventory.bloomFilter.Add([]byte(bloomKey))
	fmt.Printf("7. Added key directly to bloom filter\n")

	// Test directly
	bloomExists := inventory.bloomFilter.Test([]byte(bloomKey))
	fmt.Printf("8. Direct bloom filter test: %v (should be true)\n", bloomExists)
	assert.True(t, bloomExists, "Direct bloom filter test should return true")

	// Remove directly
	inventory.bloomFilter.Remove([]byte(bloomKey))
	fmt.Printf("9. Removed key directly from bloom filter\n")

	// Test after removal
	bloomExists = inventory.bloomFilter.Test([]byte(bloomKey))
	fmt.Printf("10. Direct bloom filter test after removal: %v (should be false)\n", bloomExists)
	assert.False(t, bloomExists, "Direct bloom filter test should return false after removal")
}
