package bloom

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"

	"github.com/redis/go-redis/v9"
)

// CountingBloomFilter Redis-based counting bloom filter that supports deletion
type CountingBloomFilter struct {
	redis     redis.Cmdable
	keyPrefix string
	m         uint64 // bit array size
	k         uint8  // number of hash functions
	maxCount  uint8  // maximum count value (to prevent overflow)
}

// CountingBloomFilterConfig configuration for counting bloom filter
type CountingBloomFilterConfig struct {
	KeyPrefix        string  // Redis key prefix
	ExpectedElements uint64  // expected number of elements
	FalsePositiveRate float64 // desired false positive rate
	MaxCount         uint8   // maximum count value (default: 15, uses 4 bits)
}

// NewCountingBloomFilter creates a new counting bloom filter
func NewCountingBloomFilter(redis redis.Cmdable, config CountingBloomFilterConfig) *CountingBloomFilter {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "cbf"
	}
	if config.ExpectedElements == 0 {
		config.ExpectedElements = 1000000 // default 1M elements
	}
	if config.FalsePositiveRate == 0 {
		config.FalsePositiveRate = 0.01 // default 1% false positive rate
	}
	if config.MaxCount == 0 {
		config.MaxCount = 15 // default max count (4 bits)
	}

	// Calculate optimal parameters
	m := optimalM(config.ExpectedElements, config.FalsePositiveRate)
	k := optimalK(m, config.ExpectedElements)

	return &CountingBloomFilter{
		redis:     redis,
		keyPrefix: config.KeyPrefix,
		m:         m,
		k:         k,
		maxCount:  config.MaxCount,
	}
}

// Add adds an element to the counting bloom filter
func (cbf *CountingBloomFilter) Add(ctx context.Context, element string) error {
	hashes := cbf.getHashes(element)
	
	// For cluster mode compatibility, process each hash separately
	for _, hash := range hashes {
		bucketIndex := (hash % cbf.m) / 8
		bitOffset := (hash % cbf.m) % 8
		bucketKey := fmt.Sprintf("%s:%d", cbf.keyPrefix, bucketIndex)
		
		// Use individual operations instead of Lua script for cluster compatibility
		script := `
			local bucket_key = KEYS[1]
			local bit_offset = tonumber(ARGV[1])
			local max_count = tonumber(ARGV[2])
			
			-- Get current counter value (4 bits per counter)
			local byte_val = redis.call('GETBIT', bucket_key, bit_offset * 4) * 8 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 1) * 4 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 2) * 2 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 3)
			
			-- Increment counter if not at max
			if byte_val < max_count then
				byte_val = byte_val + 1
				
				-- Set the 4-bit counter value
				redis.call('SETBIT', bucket_key, bit_offset * 4, math.floor(byte_val / 8) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 1, math.floor(byte_val / 4) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 2, math.floor(byte_val / 2) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 3, byte_val % 2)
			end
			
			return "OK"
		`
		
		err := cbf.redis.Eval(ctx, script, []string{bucketKey}, bitOffset, cbf.maxCount).Err()
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Remove removes an element from the counting bloom filter
func (cbf *CountingBloomFilter) Remove(ctx context.Context, element string) error {
	hashes := cbf.getHashes(element)
	
	// For cluster mode compatibility, process each hash separately
	for _, hash := range hashes {
		bucketIndex := (hash % cbf.m) / 8
		bitOffset := (hash % cbf.m) % 8
		bucketKey := fmt.Sprintf("%s:%d", cbf.keyPrefix, bucketIndex)
		
		// Use individual operations instead of Lua script for cluster compatibility
		script := `
			local bucket_key = KEYS[1]
			local bit_offset = tonumber(ARGV[1])
			
			-- Get current counter value (4 bits per counter)
			local byte_val = redis.call('GETBIT', bucket_key, bit_offset * 4) * 8 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 1) * 4 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 2) * 2 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 3)
			
			-- Decrement counter if greater than 0
			if byte_val > 0 then
				byte_val = byte_val - 1
				
				-- Set the 4-bit counter value
				redis.call('SETBIT', bucket_key, bit_offset * 4, math.floor(byte_val / 8) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 1, math.floor(byte_val / 4) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 2, math.floor(byte_val / 2) % 2)
				redis.call('SETBIT', bucket_key, bit_offset * 4 + 3, byte_val % 2)
			end
			
			return "OK"
		`
		
		err := cbf.redis.Eval(ctx, script, []string{bucketKey}, bitOffset).Err()
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Test checks if an element might be in the counting bloom filter
func (cbf *CountingBloomFilter) Test(ctx context.Context, element string) (bool, error) {
	hashes := cbf.getHashes(element)
	
	// For cluster mode compatibility, process each hash separately
	for _, hash := range hashes {
		bucketIndex := (hash % cbf.m) / 8
		bitOffset := (hash % cbf.m) % 8
		bucketKey := fmt.Sprintf("%s:%d", cbf.keyPrefix, bucketIndex)
		
		// Use individual operations instead of Lua script for cluster compatibility
		script := `
			local bucket_key = KEYS[1]
			local bit_offset = tonumber(ARGV[1])
			
			-- Get current counter value (4 bits per counter)
			local byte_val = redis.call('GETBIT', bucket_key, bit_offset * 4) * 8 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 1) * 4 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 2) * 2 +
							redis.call('GETBIT', bucket_key, bit_offset * 4 + 3)
			
			-- If any counter is 0, element is definitely not in the set
			if byte_val == 0 then
				return 0
			end
			
			return 1
		`
		
		result, err := cbf.redis.Eval(ctx, script, []string{bucketKey}, bitOffset).Int()
		if err != nil {
			return false, err
		}
		
		// If any counter is 0, element is definitely not in the set
		if result == 0 {
			return false, nil
		}
	}
	
	// All counters are > 0, element might be in the set
	return true, nil
}

// Clear clears all elements from the counting bloom filter
func (cbf *CountingBloomFilter) Clear(ctx context.Context) error {
	// Delete all keys with the prefix
	pattern := cbf.keyPrefix + ":*"
	keys, err := cbf.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return cbf.redis.Del(ctx, keys...).Err()
	}
	
	return nil
}

// getHashes generates k hash values for the given element
func (cbf *CountingBloomFilter) getHashes(element string) []uint64 {
	hashes := make([]uint64, cbf.k)
	
	// Use FNV hash as base
	h1 := fnv.New64a()
	h1.Write([]byte(element))
	hash1 := h1.Sum64()
	
	// Use a different hash function for the second hash
	h2 := fnv.New64()
	h2.Write([]byte(element + "salt"))
	hash2 := h2.Sum64()
	
	// Generate k hashes using double hashing
	for i := uint8(0); i < cbf.k; i++ {
		hashes[i] = hash1 + uint64(i)*hash2
	}
	
	return hashes
}

// optimalM calculates the optimal bit array size
func optimalM(n uint64, p float64) uint64 {
	return uint64(math.Ceil(-float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))))
}

// optimalK calculates the optimal number of hash functions
func optimalK(m, n uint64) uint8 {
	k := math.Ceil(float64(m) / float64(n) * math.Log(2))
	if k > 255 {
		k = 255
	}
	return uint8(k)
}

// Stats returns statistics about the counting bloom filter
func (cbf *CountingBloomFilter) Stats(ctx context.Context) (map[string]interface{}, error) {
	// Get all keys with the prefix
	pattern := cbf.keyPrefix + ":*"
	keys, err := cbf.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"bit_array_size":     cbf.m,
		"hash_functions":     cbf.k,
		"max_count":          cbf.maxCount,
		"redis_keys_count":   len(keys),
		"estimated_memory":   fmt.Sprintf("%.2f KB", float64(cbf.m)/8/1024),
	}
	
	return stats, nil
}