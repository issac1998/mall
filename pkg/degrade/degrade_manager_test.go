package degrade

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func TestDegradeManager_EnableDisable(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	dm := NewDegradeManager(client)
	ctx := context.Background()
	activityID := uint64(1)

	// Initially not degraded
	assert.False(t, dm.IsDegrade(ctx, activityID))

	// Enable degradation
	strategy := &DegradeStrategy{
		Type:    "queue_only",
		Message: "High traffic, queuing",
	}
	err := dm.EnableDegrade(ctx, activityID, strategy)
	assert.NoError(t, err)

	// Should be degraded now
	assert.True(t, dm.IsDegrade(ctx, activityID))

	// Get strategy
	retrievedStrategy := dm.GetStrategy(ctx, activityID)
	assert.Equal(t, "queue_only", retrievedStrategy.Type)
	assert.Equal(t, "High traffic, queuing", retrievedStrategy.Message)

	// Disable degradation
	err = dm.DisableDegrade(ctx, activityID)
	assert.NoError(t, err)

	// Should not be degraded
	assert.False(t, dm.IsDegrade(ctx, activityID))
}

func TestDegradeManager_TemporaryDegrade(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	dm := NewDegradeManager(client)
	ctx := context.Background()
	activityID := uint64(2)

	// Set temporary degradation
	strategy := &DegradeStrategy{
		Type:    "return_error",
		Message: "Temporary maintenance",
	}
	err := dm.SetTemporaryDegrade(ctx, activityID, strategy, 1*time.Second)
	assert.NoError(t, err)

	// Should be degraded
	assert.True(t, dm.IsDegrade(ctx, activityID))

	// Manually disable to simulate expiration
	err = dm.DisableDegrade(ctx, activityID)
	assert.NoError(t, err)

	// Should not be degraded anymore
	assert.False(t, dm.IsDegrade(ctx, activityID))
}

func TestDegradeManager_GetDegradeStatus(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	dm := NewDegradeManager(client)
	ctx := context.Background()

	// Enable degradation for multiple activities
	strategy1 := &DegradeStrategy{Type: "queue_only", Message: "Queue 1"}
	strategy2 := &DegradeStrategy{Type: "return_error", Message: "Error 2"}

	dm.EnableDegrade(ctx, 1, strategy1)
	dm.EnableDegrade(ctx, 2, strategy2)

	// Get all degrade status
	statuses, err := dm.GetDegradeStatus(ctx)
	assert.NoError(t, err)
	assert.Len(t, statuses, 2)

	assert.Equal(t, "queue_only", statuses[1].Type)
	assert.Equal(t, "return_error", statuses[2].Type)
}

func TestDegradeManager_DefaultStrategy(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	dm := NewDegradeManager(client)
	ctx := context.Background()
	activityID := uint64(999)

	// Get strategy for non-existent activity
	strategy := dm.GetStrategy(ctx, activityID)
	assert.Equal(t, "return_error", strategy.Type)
	assert.Equal(t, "System busy, please try again later", strategy.Message)
}

