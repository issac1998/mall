package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	// Epoch is set to the twitter snowflake epoch of Nov 04 2010 01:42:54 UTC in milliseconds
	// You may customize this to set a different epoch for your application.
	Epoch int64 = 1288834974657

	// NodeBits holds the number of bits to use for Node
	// Remember, you have a total 22 bits to share between Node/Step
	NodeBits uint8 = 10

	// StepBits holds the number of bits to use for Step
	// Remember, you have a total 22 bits to share between Node/Step
	StepBits uint8 = 12

	nodeMask = -1 ^ (-1 << NodeBits)
	stepMask = -1 ^ (-1 << StepBits)
	timeShift = NodeBits + StepBits
	nodeShift = StepBits
)

// IDGenerator ID generator using snowflake algorithm
type IDGenerator struct {
	mu        sync.Mutex
	timestamp int64
	nodeID    int64
	step      int64
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator(nodeID int64) (*IDGenerator, error) {
	if nodeID < 0 || nodeID > nodeMask {
		return nil, errors.New("invalid node ID")
	}

	return &IDGenerator{
		timestamp: 0,
		nodeID:    nodeID,
		step:      0,
	}, nil
}

// NextID generates a new ID
func (g *IDGenerator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixNano() / 1000000

	if g.timestamp == now {
		g.step = (g.step + 1) & stepMask

		if g.step == 0 {
			// Sequence exhausted, wait for next millisecond
			for now <= g.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		g.step = 0
	}

	g.timestamp = now

	id := ((now - Epoch) << timeShift) |
		(g.nodeID << nodeShift) |
		g.step

	return id
}

// ParseID parses an ID to extract timestamp, node ID and step
func ParseID(id int64) (timestamp int64, nodeID int64, step int64) {
	step = id & stepMask
	nodeID = (id >> nodeShift) & nodeMask
	timestamp = (id >> timeShift) + Epoch
	return
}

// GetTimestamp returns the timestamp part of an ID
func GetTimestamp(id int64) int64 {
	return (id >> timeShift) + Epoch
}

// GetNodeID returns the node ID part of an ID
func GetNodeID(id int64) int64 {
	return (id >> nodeShift) & nodeMask
}

// GetStep returns the step part of an ID
func GetStep(id int64) int64 {
	return id & stepMask
}

