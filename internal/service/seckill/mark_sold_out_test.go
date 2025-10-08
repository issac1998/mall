package seckill

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkSoldOut_BloomFilterRemoval(t *testing.T) {
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

	// Add activity to bloom filter first (using activityID as goodsID for testing)
	inventory.AddToBloomFilter(activityID)

	// Verify activity exists in bloom filter (through LocalCheck)
	exists := inventory.LocalCheck(ctx, activityID)
	assert.True(t, exists, "Activity should exist in bloom filter before marking sold out")

	// Mark activity as sold out
	err = inventory.MarkSoldOut(activityID)
	require.NoError(t, err)

	// Verify activity is marked as sold out in local cache
	exists = inventory.LocalCheck(ctx, activityID)
	assert.False(t, exists, "Activity should be marked as sold out after MarkSoldOut")

	// Test with a new activity that was never added to bloom filter
	newActivityID := uint64(67890)

	// This should return false (not in bloom filter)
	exists = inventory.LocalCheck(ctx, newActivityID)
	assert.False(t, exists, "New activity should not exist in bloom filter")

	// Mark it as sold out
	err = inventory.MarkSoldOut(newActivityID)
	require.NoError(t, err)

	// Should still return false (sold out)
	exists = inventory.LocalCheck(ctx, newActivityID)
	assert.False(t, exists, "New activity should remain false after MarkSoldOut")
}

func TestMarkSoldOut_BloomFilterConsistency(t *testing.T) {
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

	// Test multiple activities
	activityIDs := []uint64{1001, 1002, 1003, 1004, 1005}

	// Add all activities to bloom filter (using activityID as goodsID for testing)
	for _, activityID := range activityIDs {
		inventory.AddToBloomFilter(activityID)
	}

	// Verify all activities exist in bloom filter
	for _, activityID := range activityIDs {
		exists := inventory.LocalCheck(ctx, activityID)
		assert.True(t, exists, "Activity %d should exist in bloom filter", activityID)
	}

	// Mark some activities as sold out
	soldOutActivities := []uint64{1001, 1003, 1005}
	for _, activityID := range soldOutActivities {
		err = inventory.MarkSoldOut(activityID)
		require.NoError(t, err)
	}

	// Verify sold out activities return false
	for _, activityID := range soldOutActivities {
		exists := inventory.LocalCheck(ctx, activityID)
		assert.False(t, exists, "Sold out activity %d should return false", activityID)
	}

	// Verify remaining activities still return true
	remainingActivities := []uint64{1002, 1004}
	for _, activityID := range remainingActivities {
		exists := inventory.LocalCheck(ctx, activityID)
		assert.True(t, exists, "Remaining activity %d should still exist", activityID)
	}
}

func BenchmarkMarkSoldOut_WithBloomFilter(b *testing.B) {
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
	require.NoError(b, err)

	// Pre-populate bloom filter
	for i := 0; i < 1000; i++ {
		inventory.AddToBloomFilter(uint64(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		activityID := uint64(0)
		for pb.Next() {
			activityID = (activityID + 1) % 1000
			inventory.MarkSoldOut(activityID)
		}
	})
}
