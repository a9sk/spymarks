package core

import (
	"testing"
)

func TestPayloadFromMessageDeterministic(t *testing.T) {
	a := PayloadFromMessage("hello world")
	b := PayloadFromMessage("hello world")
	if a.Hex() != b.Hex() {
		t.Fatal("same message produced different payloads")
	}
}

func TestPayloadFromHex(t *testing.T) {
	p, err := PayloadFromHex("a5f0")
	if err != nil {
		t.Fatal(err)
	}
	if p.Hex() != "a5f0" {
		t.Fatalf("unexpected hex: %s", p.Hex())
	}
}

func TestEncodeDecodeRepeat(t *testing.T) {
	bits := []bool{true, false, true, true}
	encoded := EncodeRepeat(bits, 3)
	if len(encoded) != 12 {
		t.Fatalf("expected 12 bits, got %d", len(encoded))
	}
	decoded := DecodeRepeat(encoded, 3)
	if len(decoded) != 4 {
		t.Fatalf("expected 4 bits, got %d", len(decoded))
	}
	for i := range bits {
		if decoded[i] != bits[i] {
			t.Fatalf("bit %d mismatch", i)
		}
	}
}

func TestRepeatMajorityVote(t *testing.T) {
	// 1,1,0 should decode to 1.
	encoded := []bool{true, true, false, false, true, true}
	decoded := DecodeRepeat(encoded, 3)
	if len(decoded) != 2 {
		t.Fatalf("expected 2 bits, got %d", len(decoded))
	}
	if !decoded[0] || !decoded[1] {
		t.Fatal("majority vote failed")
	}
}

func TestUint64RoundTrip(t *testing.T) {
	v := uint64(0xA5A5A5A5A5A5A5A5)
	p := PayloadFromUint64(v)
	if p.Uint64() != v {
		t.Fatalf("uint64 round-trip failed: got %x", p.Uint64())
	}
}

func TestPayloadFromMessageBits(t *testing.T) {
	p := PayloadFromMessageBits("hello", 32)
	if p.Len() != 32 {
		t.Fatalf("expected 32 bits, got %d", p.Len())
	}
	a := PayloadFromMessageBits("hello", 32)
	b := PayloadFromMessageBits("hello", 32)
	if a.Hex() != b.Hex() {
		t.Fatal("same message and length produced different payloads")
	}
}

func TestWithSyncBits(t *testing.T) {
	p := PayloadFromUint64(0x123456789ABCDEF0)
	bits := p.WithSyncBits(16)
	if len(bits) != 16+p.Len() {
		t.Fatalf("expected %d bits, got %d", 16+p.Len(), len(bits))
	}
}
