package internal

import "testing"

func TestListMethods(t *testing.T) {
	methods := ListMethods()
	if len(methods) == 0 {
		t.Fatal("no methods listed")
	}
	seen := make(map[string]bool)
	for _, m := range methods {
		seen[m] = true
	}
	for _, want := range []string{"auto", "dct", "lsb", "fft", "text"} {
		if !seen[want] {
			t.Fatalf("missing method %s", want)
		}
	}
}

func TestResolveProviderOnlyImages(t *testing.T) {
	opts := &Options{OnlyImages: true}
	p, err := resolveProvider("auto", "audio/wav", ".wav", opts)
	if err == nil {
		t.Fatalf("expected no image provider for audio, got %s", p.Method())
	}
}

func TestResolveProviderOnlyAudio(t *testing.T) {
	opts := &Options{OnlyAudio: true}
	p, err := resolveProvider("auto", "image/png", ".png", opts)
	if err == nil {
		t.Fatalf("expected no audio provider for image, got %s", p.Method())
	}
}

func TestDetectType(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.wav", "audio/wav"},
		{"test.flac", "audio/flac"},
		{"test.txt", "text/plain"},
	}
	for _, c := range cases {
		got, err := DetectType(c.path)
		if err != nil {
			t.Fatalf("DetectType(%s): %v", c.path, err)
		}
		if got != c.want {
			t.Fatalf("DetectType(%s) = %s, want %s", c.path, got, c.want)
		}
	}
}
