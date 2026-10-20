package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const (
	demoKey   = "insecure-demo-key-do-not-use"
	hkdfSalt  = "spymarks"
	hkdfInfo  = "spy-v1"
	seedBytes = 32
)

// KeyDerive reads a key from a file or uses the raw string, then derives a
// deterministic 256-bit seed via HKDF-SHA256.
func KeyDerive(keyInput string) ([]byte, error) {
	material := []byte(demoKey)
	if keyInput != "" {
		if _, err := os.Stat(keyInput); err == nil {
			data, err := os.ReadFile(keyInput)
			if err != nil {
				return nil, fmt.Errorf("reading key file: %w", err)
			}
			material = data
		} else {
			material = []byte(keyInput)
		}
	}
	material = []byte(strings.TrimSpace(string(material)))

	h := hkdf.New(sha256.New, material, []byte(hkdfSalt), []byte(hkdfInfo))
	seed := make([]byte, seedBytes)
	if _, err := h.Read(seed); err != nil {
		return nil, fmt.Errorf("hkdf read: %w", err)
	}
	return seed, nil
}

// DeterministicReader is a deterministic byte source derived from a seed.
// It uses HMAC-SHA256 in counter mode and is suitable for reproducible
// pseudo-random choices. It is not a general-purpose CSPRNG.
type DeterministicReader struct {
	seed []byte
	ctr  uint64
}

// NewDeterministicReader creates a deterministic reader from seed material.
func NewDeterministicReader(seed []byte) *DeterministicReader {
	// Defensive copy.
	s := make([]byte, len(seed))
	copy(s, seed)
	return &DeterministicReader{seed: s}
}

// Read fills p with deterministic pseudo-random bytes.
func (r *DeterministicReader) Read(p []byte) (int, error) {
	need := len(p)
	buf := make([]byte, 0, need+hmacSHA256Size)
	for len(buf) < need {
		ctrBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(ctrBytes, r.ctr)
		mac := hmac.New(sha256.New, r.seed)
		mac.Write(ctrBytes)
		buf = append(buf, mac.Sum(nil)...)
		r.ctr++
	}
	copy(p, buf[:need])
	return need, nil
}

const hmacSHA256Size = sha256.Size

// IntN returns a uniform int in [0, n) using rejection sampling.
func (r *DeterministicReader) IntN(n int) int {
	if n <= 0 {
		panic("IntN: n must be positive")
	}
	// Use 8 bytes and reduce; good enough for block/coeff selection.
	var b [8]byte
	for {
		_, _ = r.Read(b[:])
		v := binary.BigEndian.Uint64(b[:])
		limit := ^uint64(0) - (^uint64(0))%uint64(n)
		if v < limit {
			return int(v % uint64(n))
		}
	}
}

// Bool returns a pseudo-random boolean.
func (r *DeterministicReader) Bool() bool {
	var b [1]byte
	_, _ = r.Read(b[:])
	return b[0]&1 == 1
}

// Float64 returns a pseudo-random float64 in [0,1).
func (r *DeterministicReader) Float64() float64 {
	var b [8]byte
	_, _ = r.Read(b[:])
	v := binary.BigEndian.Uint64(b[:])
	// From math/rand/v2 algorithm.
	return float64(v>>11) * (1.0 / (1 << 53))
}
