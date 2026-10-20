package core

import "fmt"

// ECC defines a pluggable error-correction code.
type ECC interface {
	// Name returns the canonical name used by CLI flags.
	Name() string
	// Encode returns an ECC-encoded bit stream.
	Encode(bits []bool) []bool
	// Decode returns the original bit stream from an ECC-encoded stream.
	Decode(bits []bool) []bool
	// Expansion returns the ratio of encoded length to original length.
	Expansion() float64
}

var eccRegistry = map[string]ECC{
	"none":    &NoneECC{},
	"repeat":  &RepeatECC{Factor: RepeatFactor},
	"hamming": &HammingECC{},
}

// GetECC returns the ECC implementation for name, or an error.
func GetECC(name string) (ECC, error) {
	if name == "" {
		name = "repeat"
	}
	ecc, ok := eccRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown ECC: %s", name)
	}
	return ecc, nil
}

// ListECC returns the names of registered ECC schemes.
func ListECC() []string {
	out := make([]string, 0, len(eccRegistry))
	for name := range eccRegistry {
		out = append(out, name)
	}
	return out
}

// NoneECC is a pass-through ECC (no redundancy).
type NoneECC struct{}

func (e *NoneECC) Name() string           { return "none" }
func (e *NoneECC) Encode(b []bool) []bool { return append([]bool(nil), b...) }
func (e *NoneECC) Decode(b []bool) []bool { return append([]bool(nil), b...) }
func (e *NoneECC) Expansion() float64     { return 1.0 }

// RepeatECC repeats each bit factor times.
type RepeatECC struct {
	Factor int
}

func (e *RepeatECC) Name() string { return "repeat" }

func (e *RepeatECC) Encode(bits []bool) []bool {
	return EncodeRepeat(bits, e.Factor)
}

func (e *RepeatECC) Decode(bits []bool) []bool {
	return DecodeRepeat(bits, e.Factor)
}

func (e *RepeatECC) Expansion() float64 {
	return float64(e.Factor)
}

// hammingEncodeNibble encodes a 4-bit nibble into 7 bits using Hamming(7,4).
// Bits are MSB-first within the nibble and within the codeword.
func hammingEncodeNibble(nib [4]bool) [7]bool {
	// Data bits d1..d4 map to positions 3,5,6,7 (1-indexed).
	// Let p1 = d1 xor d2 xor d4  -> position 1
	// p2 = d1 xor d3 xor d4      -> position 2
	// p3 = d2 xor d3 xor d4      -> position 4
	d1, d2, d3, d4 := nib[0], nib[1], nib[2], nib[3]
	p1 := d1 != d2 != d4
	p2 := d1 != d3 != d4
	p3 := d2 != d3 != d4
	// Codeword positions 1..7.
	return [7]bool{p1, p2, d1, p3, d2, d3, d4}
}

// hammingDecodeNibble decodes a 7-bit Hamming codeword to a 4-bit nibble and
// corrects a single-bit error.
func hammingDecodeNibble(cw [7]bool) [4]bool {
	// Syndrome bits.
	s1 := cw[0] != cw[2] != cw[4] != cw[6]
	s2 := cw[1] != cw[2] != cw[5] != cw[6]
	s3 := cw[3] != cw[4] != cw[5] != cw[6]
	syndrome := 0
	if s1 {
		syndrome |= 1
	}
	if s2 {
		syndrome |= 2
	}
	if s3 {
		syndrome |= 4
	}
	if syndrome != 0 {
		// Correct single-bit error at position syndrome (1-indexed).
		cw[syndrome-1] = !cw[syndrome-1]
	}
	return [4]bool{cw[2], cw[4], cw[5], cw[6]}
}

// HammingECC implements Hamming(7,4) single-error correction.
type HammingECC struct{}

func (e *HammingECC) Name() string { return "hamming" }

func (e *HammingECC) Encode(bits []bool) []bool {
	// Pad to multiple of 4.
	pad := (4 - len(bits)%4) % 4
	padded := append(append([]bool(nil), bits...), make([]bool, pad)...)
	out := make([]bool, 0, len(padded)/4*7)
	for i := 0; i < len(padded); i += 4 {
		var nib [4]bool
		copy(nib[:], padded[i:i+4])
		cw := hammingEncodeNibble(nib)
		out = append(out, cw[:]...)
	}
	return out
}

func (e *HammingECC) Decode(bits []bool) []bool {
	if len(bits)%7 != 0 {
		// Truncate to multiple of 7 to avoid panic.
		bits = bits[:len(bits)-len(bits)%7]
	}
	out := make([]bool, 0, len(bits)/7*4)
	for i := 0; i < len(bits); i += 7 {
		var cw [7]bool
		copy(cw[:], bits[i:i+7])
		nib := hammingDecodeNibble(cw)
		out = append(out, nib[:]...)
	}
	return out
}

func (e *HammingECC) Expansion() float64 { return 7.0 / 4.0 }
