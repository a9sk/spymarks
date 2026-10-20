package internal

import "github.com/a9sk/spymarks/internal/core"

// Re-export payload helpers from core for internal consumers.
type Payload = core.Payload

const (
	DefaultPayloadBits = core.DefaultPayloadBits
	DefaultSyncBits    = core.DefaultSyncBits
	DefaultSyncPattern = core.DefaultSyncPattern
	RepeatFactor       = core.RepeatFactor
)

var (
	PayloadFromMessage     = core.PayloadFromMessage
	PayloadFromMessageBits = core.PayloadFromMessageBits
	PayloadFromHex         = core.PayloadFromHex
	PayloadFromBits        = core.PayloadFromBits
	PayloadFromUint64      = core.PayloadFromUint64
	EncodeRepeat           = core.EncodeRepeat
	DecodeRepeat           = core.DecodeRepeat
)
