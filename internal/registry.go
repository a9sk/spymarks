package internal

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a9sk/spymarks/internal/providers/audio"
	"github.com/a9sk/spymarks/internal/providers/image"
	"github.com/a9sk/spymarks/internal/providers/text"
)

var providers = []Provider{
	image.NewDCTProvider(),
	image.NewLSBProvider(),
	audio.NewFFTProvider(),
	text.NewSynonymProvider(),
}

// ListMethods returns the canonical method names of all registered providers.
func ListMethods() []string {
	out := []string{"auto"}
	for _, p := range providers {
		out = append(out, p.Method())
	}
	return out
}

// providerFilter returns true if p should be considered under opts.
func providerFilter(p Provider, opts *Options) bool {
	if opts == nil {
		return true
	}
	if opts.OnlyImages && !isImageProvider(p) {
		return false
	}
	if opts.OnlyAudio && !isAudioProvider(p) {
		return false
	}
	if opts.OnlyText && !isTextProvider(p) {
		return false
	}
	return true
}

func isImageProvider(p Provider) bool {
	return p.Method() == "dct" || p.Method() == "lsb"
}

func isAudioProvider(p Provider) bool {
	return p.Method() == "fft"
}

func isTextProvider(p Provider) bool {
	return p.Method() == "text"
}

// DetectType returns a MIME-like type string based on extension and optional
// content sniffing.
func DetectType(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".flac":
		return "audio/flac", nil
	case ".txt":
		return "text/plain", nil
	default:
		// Basic sniff.
		b, err := readFileHeader(path, 512)
		if err != nil {
			return "", err
		}
		if len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n" {
			return "image/png", nil
		}
		if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xD8 {
			return "image/jpeg", nil
		}
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// resolveProvider selects a provider based on method and file type.
func resolveProvider(method, mime, ext string, opts *Options) (Provider, error) {
	if method != "" && method != "auto" {
		for _, p := range providers {
			if p.Method() == method && providerFilter(p, opts) {
				return p, nil
			}
		}
		return nil, fmt.Errorf("unknown method: %s", method)
	}
	for _, p := range providers {
		if p.Supported(mime, ext) && providerFilter(p, opts) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no provider supports %s", mime)
}

// resolveProviders returns all providers that support the given file type.
func resolveProviders(method, mime, ext string, opts *Options) ([]Provider, error) {
	if method != "" && method != "auto" {
		for _, p := range providers {
			if p.Method() == method && providerFilter(p, opts) {
				return []Provider{p}, nil
			}
		}
		return nil, fmt.Errorf("unknown method: %s", method)
	}
	var out []Provider
	for _, p := range providers {
		if p.Supported(mime, ext) && providerFilter(p, opts) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no provider supports %s", mime)
	}
	return out, nil
}
