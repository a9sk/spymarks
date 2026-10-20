package internal

import (
	"fmt"
	"path/filepath"
)

// Check detects a spymark in inputPath and returns the detection result.
func Check(inputPath string, opts *Options) (*CheckResult, error) {
	mime, err := DetectType(inputPath)
	if err != nil {
		return nil, fmt.Errorf("detecting type: %w", err)
	}

	providers, err := resolveProviders(opts.Method, mime, filepath.Ext(inputPath), opts)
	if err != nil {
		return nil, fmt.Errorf("resolving provider: %w", err)
	}

	seed, err := KeyDerive(opts.Key)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	cfg, err := ToEmbedConfig(opts)
	if err != nil {
		return nil, err
	}

	var payload *Payload
	if opts.RawHex != "" {
		payload, err = PayloadFromHex(opts.RawHex)
		if err != nil {
			return nil, err
		}
	} else {
		payload = PayloadFromMessageBits(opts.Message, cfg.PayloadBits)
	}

	var best *CheckResult
	for _, provider := range providers {
		res, err := provider.Check(inputPath, payload.Len(), seed, cfg)
		if err != nil {
			return nil, fmt.Errorf("checking mark with %s: %w", provider.Method(), err)
		}
		if best == nil || res.Confidence > best.Confidence {
			best = res
		}
	}
	best.Filename = inputPath
	return best, nil
}
