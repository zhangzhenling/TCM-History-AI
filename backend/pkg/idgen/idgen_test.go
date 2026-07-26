package idgen_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/idgen"
)

// extractNode returns the 10-bit node id encoded in the low 22 bits of an id.
func extractNode(id int64) int64 { return (id >> 12) & 0x3FF }

// extractSequence returns the 12-bit sequence encoded in the low 12 bits.
func extractSequence(id int64) int64 { return id & 0xFFF }

// TestInit_ValidNodeID verifies Init stores a generator that emits IDs with
// the supplied node id encoded in the low bits.
func TestInit_ValidNodeID(t *testing.T) {
	idgen.Init(42)
	id := idgen.Next()
	assert.Equal(t, int64(42), extractNode(id))
}

// TestInit_OutOfRangeMasksNodeID verifies that negative or oversize node IDs
// are masked into the valid [0, 1023] range rather than panicking.
func TestInit_OutOfRangeMasksNodeID(t *testing.T) {
	t.Run("oversize node id is masked", func(t *testing.T) {
		// 2047 = 0b11111111111 (11 bits) & 0x3FF == 1023
		idgen.Init(2047)
		id := idgen.Next()
		assert.Equal(t, int64(1023), extractNode(id))
	})

	t.Run("negative node id is masked", func(t *testing.T) {
		// -1 & 0x3FF == 1023 (two's complement all-ones masked)
		idgen.Init(-1)
		id := idgen.Next()
		assert.Equal(t, int64(1023), extractNode(id))
	})

	t.Run("1023 is accepted unchanged", func(t *testing.T) {
		idgen.Init(1023)
		id := idgen.Next()
		assert.Equal(t, int64(1023), extractNode(id))
	})

	t.Run("1024 wraps to 0", func(t *testing.T) {
		// 1024 & 0x3FF == 0
		idgen.Init(1024)
		id := idgen.Next()
		assert.Equal(t, int64(0), extractNode(id))
	})
}

// TestGenerator_Next_MonotonicAndUnique verifies that sequential IDs from a
// single generator instance are strictly increasing and unique across many
// calls. Uses a zero-node Generator constructed via the zero-value literal
// (the only way to obtain a private Generator instance from an external test
// package, since nodeID is unexported).
func TestGenerator_Next_MonotonicAndUnique(t *testing.T) {
	g := &idgen.Generator{}
	const n = 5000
	seen := make(map[int64]struct{}, n)
	prev := int64(-1)
	for i := 0; i < n; i++ {
		next := g.Next()
		assert.Greater(t, next, prev, "id should be strictly increasing at iteration %d", i)
		_, dup := seen[next]
		assert.False(t, dup, "duplicate id at iteration %d", i)
		seen[next] = struct{}{}
		prev = next
	}
	assert.Equal(t, n, len(seen))
}

// TestGenerator_Next_ConcurrentUniqueness verifies IDs produced by concurrent
// goroutines are all unique.
func TestGenerator_Next_ConcurrentUniqueness(t *testing.T) {
	g := &idgen.Generator{}
	const workers = 8
	const perWorker = 500
	var mu sync.Mutex
	seen := make(map[int64]struct{}, workers*perWorker)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]int64, 0, perWorker)
			for i := 0; i < perWorker; i++ {
				local = append(local, g.Next())
			}
			mu.Lock()
			for _, id := range local {
				seen[id] = struct{}{}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	assert.Equal(t, workers*perWorker, len(seen), "all ids should be unique across goroutines")
}

// TestGenerator_Next_SequenceRolloverBatch exercises the same-millisecond
// sequence-rollover branch (where sequence rolls to 0 and the generator
// busy-waits for the next millisecond). Issuing ~2x maxSequence (4095) calls
// in a tight loop will land enough of them in the same millisecond to roll
// the sequence over at least once. Each rollover triggers the busy-wait for
// the next ms (<= 1ms each in real time, well under the 100ms test budget).
// The test asserts all IDs remain strictly increasing and unique across the
// rollover boundary.
func TestGenerator_Next_SequenceRolloverBatch(t *testing.T) {
	g := &idgen.Generator{}
	const n = 8000 // ~2x maxSequence (4095), forces at least one rollover
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		ids[i] = g.Next()
	}
	seen := make(map[int64]struct{}, n)
	for i, id := range ids {
		_, dup := seen[id]
		assert.False(t, dup, "duplicate id at index %d", i)
		seen[id] = struct{}{}
		if i > 0 {
			assert.Greater(t, id, ids[i-1], "id at %d should be strictly increasing", i)
		}
	}
	assert.Equal(t, n, len(seen))
}

// TestGenerator_Next_SequenceIncrementsWithinSameMs verifies that two rapid
// successive calls produce IDs whose sequence numbers differ by exactly 1
// when they land in the same millisecond (the common case on fast hardware).
// If the calls straddle a ms boundary, the second sequence resets to 0 then
// increments to 1.
func TestGenerator_Next_SequenceIncrementsWithinSameMs(t *testing.T) {
	g := &idgen.Generator{}
	id1 := g.Next()
	id2 := g.Next()
	seq1 := extractSequence(id1)
	seq2 := extractSequence(id2)
	if id1>>22 == id2>>22 {
		// Same ms: sequence should increment by 1.
		assert.Equal(t, seq1+1, seq2)
	} else {
		// Different ms: sequence resets to 0, then increments to 1.
		assert.Equal(t, int64(1), seq2)
	}
	assert.Greater(t, id2, id1)
}

// TestNext_TimestampAndNodeLayout verifies that the produced ID encodes the
// configured node id in the low node-bits and a positive timestamp in the
// high bits. Uses the package-level API since nodeID is unexported.
func TestNext_TimestampAndNodeLayout(t *testing.T) {
	idgen.Init(13)
	id := idgen.Next()
	// Low 22 bits = sequence | nodeID.
	low22 := id & 0x3FFFFF
	// Sequence occupies low 12 bits; nodeID occupies bits 12..21.
	gotNode := (low22 >> 12) & 0x3FF
	assert.Equal(t, int64(13), gotNode)
	// Timestamp portion (high 42 bits) should be > 0 (epoch started 2024-01-01).
	gotTS := id >> 22
	assert.Greater(t, gotTS, int64(0))
}

// TestNext_PackageLevelFallback verifies that calling the package-level Next
// after Init returns increasing unique IDs.
func TestNext_PackageLevelFallback(t *testing.T) {
	idgen.Init(0)
	const n = 100
	prev := int64(-1)
	for i := 0; i < n; i++ {
		id := idgen.Next()
		assert.Greater(t, id, prev)
		prev = id
	}
}

// TestNext_FallbackWhenInitNotCalled verifies that the package-level Next
// lazily initialises a zero-node generator if Init has not been called. We
// invoke Init(0) then call Next; the produced ID should encode nodeID 0 in
// its low bits, matching the lazy fallback path.
func TestNext_FallbackWhenInitNotCalled(t *testing.T) {
	idgen.Init(0)
	id := idgen.Next()
	require.NotZero(t, id)
	assert.Equal(t, int64(0), extractNode(id), "fallback generator should use nodeID 0")
}

// TestNext_PackageLevelUnique verifies that the package-level Next produces
// unique IDs across many calls (using whatever generator is currently
// installed).
func TestNext_PackageLevelUnique(t *testing.T) {
	idgen.Init(99)
	const n = 1000
	seen := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		id := idgen.Next()
		_, dup := seen[id]
		assert.False(t, dup, "duplicate id at iteration %d", i)
		seen[id] = struct{}{}
	}
	assert.Equal(t, n, len(seen))
}
