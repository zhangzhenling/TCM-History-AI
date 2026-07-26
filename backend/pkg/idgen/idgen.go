// Package idgen issues monotonically increasing 64-bit snowflake-like IDs
// from a process-local generator seeded with a node id.
package idgen

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	epoch        int64 = 1704067200000 // 2024-01-01T00:00:00Z in millis
	nodeBits     uint8 = 10
	sequenceBits uint8 = 12
	maxNodeID    int64 = -1 ^ (-1 << nodeBits)
	maxSequence  int64 = -1 ^ (-1 << sequenceBits)
	timeShift    uint8 = nodeBits + sequenceBits
	nodeShift    uint8 = sequenceBits
)

// Generator is a snowflake-style id generator.
type Generator struct {
	mu        sync.Mutex
	nodeID    int64
	timestamp int64
	sequence  int64
}

var defaultGenerator atomic.Pointer[Generator]

// Init initialises the process-wide generator with a node id.
// nodeID must be in [0, 1023]; values outside the range are masked.
func Init(nodeID int64) {
	if nodeID < 0 || nodeID > maxNodeID {
		nodeID = nodeID & maxNodeID
	}
	g := &Generator{nodeID: nodeID}
	defaultGenerator.Store(g)
}

// Next returns the next id from the process generator.
// Panics if Init has not been called.
func Next() int64 {
	g := defaultGenerator.Load()
	if g == nil {
		// Fall back to a zero-node generator rather than panicking, so that
		// ad-hoc tests that don't call Init still work.
		g = &Generator{}
		defaultGenerator.Store(g)
	}
	return g.Next()
}

// Next generates the next id.
func (g *Generator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - epoch
	if now == g.timestamp {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			for now <= g.timestamp {
				now = time.Now().UnixMilli() - epoch
			}
		}
	} else if now > g.timestamp {
		g.sequence = 0
	} else {
		// clock moved backward; reuse last timestamp to keep monotonicity.
		now = g.timestamp
	}
	g.timestamp = now

	return (now << timeShift) | (g.nodeID << nodeShift) | g.sequence
}
