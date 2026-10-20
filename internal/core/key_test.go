package core

import (
	"bytes"
	"testing"
)

func TestKeyDeriveDeterministic(t *testing.T) {
	a, err := KeyDerive("my-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	b, err := KeyDerive("my-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("key derivation is not deterministic")
	}
	if len(a) != seedBytes {
		t.Fatalf("expected %d bytes, got %d", seedBytes, len(a))
	}
}

func TestKeyDeriveDifferentKeys(t *testing.T) {
	a, _ := KeyDerive("key-a")
	b, _ := KeyDerive("key-b")
	if bytes.Equal(a, b) {
		t.Fatal("different keys produced same seed")
	}
}

func TestDeterministicReader(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")
	r1 := NewDeterministicReader(seed)
	r2 := NewDeterministicReader(seed)

	b1 := make([]byte, 64)
	b2 := make([]byte, 64)
	if _, err := r1.Read(b1); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Read(b2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("deterministic reader produced different bytes")
	}
}

func TestDeterministicReaderIntN(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")
	r := NewDeterministicReader(seed)
	for i := 0; i < 1000; i++ {
		v := r.IntN(100)
		if v < 0 || v >= 100 {
			t.Fatalf("IntN out of range: %d", v)
		}
	}
}
