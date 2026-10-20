package internal

import "github.com/a9sk/spymarks/internal/core"

// Re-export key helpers from core for internal consumers.
var (
	KeyDerive              = core.KeyDerive
	NewDeterministicReader = core.NewDeterministicReader
)

// DeterministicReader is a deterministic byte source.
type DeterministicReader = core.DeterministicReader
