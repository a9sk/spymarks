package core

import (
	"testing"
)

func TestNoneECCRoundTrip(t *testing.T) {
	ecc := &NoneECC{}
	bits := []bool{true, false, true, true, false}
	enc := ecc.Encode(bits)
	dec := ecc.Decode(enc)
	if len(dec) != len(bits) {
		t.Fatalf("length mismatch: %d vs %d", len(dec), len(bits))
	}
	for i := range bits {
		if dec[i] != bits[i] {
			t.Fatalf("bit %d mismatch", i)
		}
	}
}

func TestRepeatECCRoundTrip(t *testing.T) {
	ecc := &RepeatECC{Factor: 3}
	bits := []bool{true, false, true, true}
	enc := ecc.Encode(bits)
	if len(enc) != len(bits)*3 {
		t.Fatalf("expected encoded length %d, got %d", len(bits)*3, len(enc))
	}
	dec := ecc.Decode(enc)
	if len(dec) != len(bits) {
		t.Fatalf("expected decoded length %d, got %d", len(bits), len(dec))
	}
	for i := range bits {
		if dec[i] != bits[i] {
			t.Fatalf("bit %d mismatch", i)
		}
	}
}

func TestRepeatECCMajorityVote(t *testing.T) {
	ecc := &RepeatECC{Factor: 3}
	// One flipped bit in each triplet.
	enc := []bool{true, false, true, false, true, true}
	dec := ecc.Decode(enc)
	if len(dec) != 2 {
		t.Fatalf("expected 2 bits, got %d", len(dec))
	}
	if !dec[0] || !dec[1] {
		t.Fatal("majority vote failed")
	}
}

func TestHammingRoundTrip(t *testing.T) {
	ecc := &HammingECC{}
	bits := []bool{true, false, true, true, false, false, true, false}
	enc := ecc.Encode(bits)
	if len(enc) != 14 {
		t.Fatalf("expected encoded length 14, got %d", len(enc))
	}
	dec := ecc.Decode(enc)
	if len(dec) != len(bits) {
		t.Fatalf("expected decoded length %d, got %d", len(bits), len(dec))
	}
	for i := range bits {
		if dec[i] != bits[i] {
			t.Fatalf("bit %d mismatch", i)
		}
	}
}

func TestHammingSingleBitCorrection(t *testing.T) {
	ecc := &HammingECC{}
	bits := []bool{true, false, true, true, false, false, true, false}
	enc := ecc.Encode(bits)
	// Flip one bit in each codeword.
	for i := 0; i < len(enc); i += 7 {
		enc[i+2] = !enc[i+2]
	}
	dec := ecc.Decode(enc)
	if len(dec) != len(bits) {
		t.Fatalf("expected decoded length %d, got %d", len(bits), len(dec))
	}
	for i := range bits {
		if dec[i] != bits[i] {
			t.Fatalf("bit %d mismatch after correction", i)
		}
	}
}

func TestGetECCDefaults(t *testing.T) {
	ecc, err := GetECC("")
	if err != nil {
		t.Fatal(err)
	}
	if ecc.Name() != "repeat" {
		t.Fatalf("expected default repeat, got %s", ecc.Name())
	}
}

func TestListECC(t *testing.T) {
	names := ListECC()
	if len(names) == 0 {
		t.Fatal("no ECC schemes listed")
	}
	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	for _, want := range []string{"none", "repeat", "hamming"} {
		if !seen[want] {
			t.Fatalf("missing ECC scheme %s", want)
		}
	}
}
