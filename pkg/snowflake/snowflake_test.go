package snowflake

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDGenerator(t *testing.T) {
	t.Run("NewIDGenerator", func(t *testing.T) {
		// Valid node ID
		gen, err := NewIDGenerator(1)
		assert.NoError(t, err)
		assert.NotNil(t, gen)
		assert.Equal(t, int64(1), gen.nodeID)

		// Invalid node ID - negative
		gen, err = NewIDGenerator(-1)
		assert.Error(t, err)
		assert.Nil(t, gen)

		// Invalid node ID - too large
		gen, err = NewIDGenerator(nodeMask + 1)
		assert.Error(t, err)
		assert.Nil(t, gen)

		// Boundary values
		gen, err = NewIDGenerator(0)
		assert.NoError(t, err)
		assert.NotNil(t, gen)

		gen, err = NewIDGenerator(nodeMask)
		assert.NoError(t, err)
		assert.NotNil(t, gen)
	})

	t.Run("NextID", func(t *testing.T) {
		gen, err := NewIDGenerator(1)
		require.NoError(t, err)

		// Generate multiple IDs
		ids := make([]int64, 100)
		for i := 0; i < 100; i++ {
			ids[i] = gen.NextID()
		}

		// All IDs should be unique
		idSet := make(map[int64]bool)
		for _, id := range ids {
			assert.False(t, idSet[id], "Duplicate ID generated: %d", id)
			idSet[id] = true
		}

		// IDs should be positive
		for _, id := range ids {
			assert.Positive(t, id)
		}

		// IDs should be roughly increasing (allowing for clock adjustments)
		for i := 1; i < len(ids); i++ {
			// Extract timestamps
			ts1 := GetTimestamp(ids[i-1])
			ts2 := GetTimestamp(ids[i])
			
			// Timestamps should be equal or increasing
			assert.True(t, ts2 >= ts1, "Timestamp should not decrease")
		}
	})

	t.Run("ConcurrentGeneration", func(t *testing.T) {
		gen, err := NewIDGenerator(1)
		require.NoError(t, err)

		const numGoroutines = 10
		const idsPerGoroutine = 100
		
		var wg sync.WaitGroup
		idChan := make(chan int64, numGoroutines*idsPerGoroutine)

		// Start multiple goroutines generating IDs
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id := gen.NextID()
					idChan <- id
				}
			}()
		}

		wg.Wait()
		close(idChan)

		// Collect all IDs
		var ids []int64
		for id := range idChan {
			ids = append(ids, id)
		}

		// Verify all IDs are unique
		idSet := make(map[int64]bool)
		for _, id := range ids {
			assert.False(t, idSet[id], "Duplicate ID generated in concurrent test: %d", id)
			idSet[id] = true
		}

		assert.Equal(t, numGoroutines*idsPerGoroutine, len(ids))
	})

	t.Run("ParseID", func(t *testing.T) {
		gen, err := NewIDGenerator(123)
		require.NoError(t, err)

		id := gen.NextID()

		// Parse the ID
		timestamp, nodeID, step := ParseID(id)

		// Verify components
		assert.Equal(t, int64(123), nodeID)
		assert.GreaterOrEqual(t, step, int64(0))
		assert.LessOrEqual(t, step, int64(stepMask))
		assert.Greater(t, timestamp, Epoch)

		// Verify timestamp is recent (within last minute)
		now := time.Now().UnixNano() / 1000000
		assert.True(t, timestamp >= now-60000 && timestamp <= now+1000)
	})

	t.Run("GetTimestamp", func(t *testing.T) {
		gen, err := NewIDGenerator(1)
		require.NoError(t, err)

		beforeTime := time.Now().UnixNano() / 1000000
		id := gen.NextID()
		afterTime := time.Now().UnixNano() / 1000000

		timestamp := GetTimestamp(id)
		
		// Timestamp should be between before and after
		assert.True(t, timestamp >= beforeTime && timestamp <= afterTime)
	})

	t.Run("GetNodeID", func(t *testing.T) {
		nodeID := int64(456)
		gen, err := NewIDGenerator(nodeID)
		require.NoError(t, err)

		id := gen.NextID()
		extractedNodeID := GetNodeID(id)

		assert.Equal(t, nodeID, extractedNodeID)
	})

	t.Run("GetStep", func(t *testing.T) {
		gen, err := NewIDGenerator(1)
		require.NoError(t, err)

		// Generate multiple IDs in quick succession
		var steps []int64
		for i := 0; i < 10; i++ {
			id := gen.NextID()
			step := GetStep(id)
			steps = append(steps, step)
		}

		// Steps should be sequential (0, 1, 2, ...) if generated in same millisecond
		// or reset to 0 if in different milliseconds
		for i, step := range steps {
			assert.GreaterOrEqual(t, step, int64(0))
			assert.LessOrEqual(t, step, int64(stepMask))
			
			if i > 0 {
				// Step should either increment or reset to 0
				prevStep := steps[i-1]
				assert.True(t, step == prevStep+1 || step == 0,
					"Step should increment or reset, got %d after %d", step, prevStep)
			}
		}
	})

	t.Run("MultipleGenerators", func(t *testing.T) {
		gen1, err := NewIDGenerator(1)
		require.NoError(t, err)
		
		gen2, err := NewIDGenerator(2)
		require.NoError(t, err)

		// Generate IDs from both generators
		var ids []int64
		for i := 0; i < 100; i++ {
			ids = append(ids, gen1.NextID())
			ids = append(ids, gen2.NextID())
		}

		// All IDs should be unique
		idSet := make(map[int64]bool)
		for _, id := range ids {
			assert.False(t, idSet[id], "Duplicate ID generated across generators: %d", id)
			idSet[id] = true
		}

		// Verify node IDs are correct
		for i := 0; i < len(ids); i += 2 {
			assert.Equal(t, int64(1), GetNodeID(ids[i]))
			assert.Equal(t, int64(2), GetNodeID(ids[i+1]))
		}
	})

	t.Run("SequenceExhaustion", func(t *testing.T) {
		gen, err := NewIDGenerator(1)
		require.NoError(t, err)

		// This test is hard to reproduce reliably since it depends on timing
		// We'll generate many IDs quickly and verify they're all unique
		const numIDs = 5000
		ids := make([]int64, numIDs)
		
		for i := 0; i < numIDs; i++ {
			ids[i] = gen.NextID()
		}

		// Verify uniqueness
		idSet := make(map[int64]bool)
		for _, id := range ids {
			assert.False(t, idSet[id], "Duplicate ID in sequence exhaustion test: %d", id)
			idSet[id] = true
		}
	})

	t.Run("Constants", func(t *testing.T) {
		// Verify constants are as expected
		assert.Equal(t, int64(1288834974657), Epoch)
		assert.Equal(t, uint8(10), NodeBits)
		assert.Equal(t, uint8(12), StepBits)
		
		// Verify masks
		assert.Equal(t, int64(nodeMask), int64(1023)) // 2^10 - 1
		assert.Equal(t, int64(stepMask), int64(4095)) // 2^12 - 1
	})
}