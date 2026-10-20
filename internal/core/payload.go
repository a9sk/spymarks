package core

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultPayloadBits = 64
	DefaultSyncBits    = 32
	DefaultSyncPattern = 0xA5A5A5A5 // 32-bit alternating sync pattern
	RepeatFactor       = 3          // default repetition coding factor
)

// Payload holds a binary payload and its bit length.
type Payload struct {
	Bits []bool
}

// PayloadFromMessage creates a deterministic 64-bit payload from a message.
func PayloadFromMessage(msg string) *Payload {
	return PayloadFromMessageBits(msg, DefaultPayloadBits)
}

// PayloadFromMessageBits creates a deterministic payload of the requested
// length from a message.
func PayloadFromMessageBits(msg string, bits int) *Payload {
	if bits <= 0 {
		bits = DefaultPayloadBits
	}
	if msg == "" {
		msg = fmt.Sprintf("spy-test-pattern-%d", time.Now().Unix())
	}
	sum := sha256.Sum256([]byte(msg))
	allBits := bytesToBits(sum[:])
	if len(allBits) < bits {
		// Pad by repeating.
		for len(allBits) < bits {
			allBits = append(allBits, allBits...)
		}
	}
	return &Payload{Bits: allBits[:bits]}
}

// PayloadFromHex parses a hex string into a payload.
func PayloadFromHex(h string) (*Payload, error) {
	h = strings.TrimSpace(h)
	if len(h) >= 2 && strings.HasPrefix(h, "0x") {
		h = h[2:]
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("invalid hex payload: %w", err)
	}
	return &Payload{Bits: bytesToBits(b)}, nil
}

// PayloadFromBits builds a payload from raw bits.
func PayloadFromBits(bits []bool) *Payload {
	return &Payload{Bits: append([]bool(nil), bits...)}
}

// PayloadFromUint64 builds a 64-bit payload from an integer.
func PayloadFromUint64(v uint64) *Payload {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return &Payload{Bits: bytesToBits(b)}
}

// Hex returns the payload as a lower-case hex string.
func (p *Payload) Hex() string {
	if len(p.Bits) == 0 {
		return ""
	}
	return hex.EncodeToString(bitsToBytes(p.Bits))
}

// Uint64 returns the first 64 bits as an integer, zero-padding if needed.
func (p *Payload) Uint64() uint64 {
	b := bitsToBytes(p.Bits)
	if len(b) >= 8 {
		return binary.BigEndian.Uint64(b[:8])
	}
	pad := make([]byte, 8-len(b))
	return binary.BigEndian.Uint64(append(pad, b...))
}

// WithSync returns sync bits followed by payload bits using the default sync length.
func (p *Payload) WithSync() []bool {
	return p.WithSyncBits(DefaultSyncBits)
}

// WithSyncBits returns sync bits of length n followed by payload bits.
func (p *Payload) WithSyncBits(n int) []bool {
	if n <= 0 {
		return append([]bool(nil), p.Bits...)
	}
	sync := uintToBits(DefaultSyncPattern, n)
	out := make([]bool, 0, n+len(p.Bits))
	out = append(out, sync...)
	out = append(out, p.Bits...)
	return out
}

// SyncPattern returns the default sync pattern truncated to n bits.
func SyncPattern(n int) uint64 {
	if n >= DefaultSyncBits {
		return DefaultSyncPattern
	}
	return DefaultSyncPattern >> (DefaultSyncBits - n)
}

// Len returns the number of payload bits.
func (p *Payload) Len() int { return len(p.Bits) }

// EncodeRepeat applies repetition coding (each bit repeated factor times).
func EncodeRepeat(bits []bool, factor int) []bool {
	if factor <= 1 {
		return append([]bool(nil), bits...)
	}
	out := make([]bool, 0, len(bits)*factor)
	for _, b := range bits {
		for i := 0; i < factor; i++ {
			out = append(out, b)
		}
	}
	return out
}

// DecodeRepeat decodes repetition-coded bits using majority vote.
func DecodeRepeat(bits []bool, factor int) []bool {
	if factor <= 1 {
		return append([]bool(nil), bits...)
	}
	outLen := len(bits) / factor
	out := make([]bool, 0, outLen)
	for i := 0; i < outLen; i++ {
		ones := 0
		for j := 0; j < factor; j++ {
			if bits[i*factor+j] {
				ones++
			}
		}
		out = append(out, ones > factor/2)
	}
	return out
}

// bitsToBytes packs bits into bytes, MSB first.
func bitsToBytes(bits []bool) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b {
			out[i/8] |= 1 << (7 - i%8)
		}
	}
	return out
}

// bytesToBits unpacks bytes into bits, MSB first.
func bytesToBits(data []byte) []bool {
	out := make([]bool, 0, len(data)*8)
	for _, by := range data {
		for i := 7; i >= 0; i-- {
			out = append(out, (by>>i)&1 == 1)
		}
	}
	return out
}

// uintToBits returns the lowest n bits of v as bools, MSB first.
func uintToBits(v uint64, n int) []bool {
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = (v>>(n-1-i))&1 == 1
	}
	return out
}
