package internal

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func TestEndToEndLSB(t *testing.T) {
	path := "/tmp/spymarks-e2e-lsb.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	opts := &Options{
		Key:      "e2e-lsb-key",
		Method:   "lsb",
		Message:  "user-1234",
		Strength: 1.0,
	}

	res, err := Mark(path, opts)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(res.Output)

	checkOpts := &Options{
		Key:       "e2e-lsb-key",
		Method:    "lsb",
		Message:   "user-1234",
		Strength:  1.0,
		Threshold: 0.6,
	}
	checkRes, err := Check(res.Output, checkOpts)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !checkRes.Detected {
		t.Fatalf("expected detection, got confidence %.2f", checkRes.Confidence)
	}
	if checkRes.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s check=%s", res.Payload, checkRes.Payload)
	}
}

func TestEndToEndHamming(t *testing.T) {
	path := "/tmp/spymarks-e2e-hamming.png"
	if err := makeTestImage(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	opts := &Options{
		Key:      "e2e-hamming-key",
		Method:   "dct",
		Message:  "user-1234",
		Strength: 12.0,
		ECC:      "hamming",
	}

	res, err := Mark(path, opts)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(res.Output)

	checkOpts := &Options{
		Key:       "e2e-hamming-key",
		Method:    "dct",
		Message:   "user-1234",
		Strength:  12.0,
		Threshold: 0.6,
		ECC:       "hamming",
	}
	checkRes, err := Check(res.Output, checkOpts)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !checkRes.Detected {
		t.Fatalf("expected detection, got confidence %.2f", checkRes.Confidence)
	}
	if checkRes.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s check=%s", res.Payload, checkRes.Payload)
	}
}

func TestEndToEndAudio(t *testing.T) {
	path := "/tmp/spymarks-e2e-audio.wav"
	if err := makeTestWAV(path, 2000); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	opts := &Options{
		Key:        "e2e-audio-key",
		Method:     "fft",
		Message:    "audio-user-1234",
		Strength:   64.0,
		Iterations: 2,
	}

	res, err := Mark(path, opts)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(res.Output)

	checkOpts := &Options{
		Key:        "e2e-audio-key",
		Method:     "fft",
		Message:    "audio-user-1234",
		Strength:   64.0,
		Threshold:  0.6,
		Iterations: 2,
	}
	checkRes, err := Check(res.Output, checkOpts)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !checkRes.Detected {
		t.Fatalf("expected detection, got confidence %.2f", checkRes.Confidence)
	}
	if checkRes.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s check=%s", res.Payload, checkRes.Payload)
	}
}

func TestEndToEndText(t *testing.T) {
	path := "/tmp/spymarks-e2e-text.txt"
	text := ""
	for i := 0; i < 40; i++ {
		text += "The quick brown fox jumps over the lazy dog. A good start is to make fast progress and help everyone feel happy. "
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	opts := &Options{
		Key:         "e2e-text-key",
		Method:      "text",
		Message:     "text-test",
		Strength:    1.0,
		PayloadBits: 16,
		SyncBits:    8,
		ECC:         "none",
	}

	res, err := Mark(path, opts)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	defer os.Remove(res.Output)

	checkOpts := &Options{
		Key:         "e2e-text-key",
		Method:      "text",
		Message:     "text-test",
		Strength:    1.0,
		Threshold:   0.6,
		PayloadBits: 16,
		SyncBits:    8,
		ECC:         "none",
	}
	checkRes, err := Check(res.Output, checkOpts)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !checkRes.Detected {
		t.Fatalf("expected detection, got confidence %.2f", checkRes.Confidence)
	}
	if checkRes.Payload != res.Payload {
		t.Fatalf("payload mismatch: mark=%s check=%s", res.Payload, checkRes.Payload)
	}
}

func makeTestWAV(path string, frames int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rate := 16000
	channels := 1
	bitsPerSample := 16
	frameSize := 1024
	hop := 1024
	numSamples := frames*hop + frameSize
	dataSize := numSamples * channels * bitsPerSample / 8
	byteRate := rate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	binary.Write(f, binary.LittleEndian, []byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	binary.Write(f, binary.LittleEndian, []byte("WAVE"))
	binary.Write(f, binary.LittleEndian, []byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(channels))
	binary.Write(f, binary.LittleEndian, uint32(rate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))
	binary.Write(f, binary.LittleEndian, []byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	for i := 0; i < numSamples; i++ {
		v := int16((math.Sin(2*math.Pi*440*float64(i)/float64(rate)) + 0.5*math.Sin(2*math.Pi*880*float64(i)/float64(rate))) * 3000)
		binary.Write(f, binary.LittleEndian, v)
	}
	return nil
}
