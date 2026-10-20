package internal

import (
	"fmt"
	"path/filepath"
)

// Mark embeds a spymark into inputPath and writes the result to the configured
// output path.
func Mark(inputPath string, opts *Options) (*MarkResult, error) {
	mime, err := DetectType(inputPath)
	if err != nil {
		return nil, fmt.Errorf("detecting type: %w", err)
	}

	provider, err := resolveProvider(opts.Method, mime, filepath.Ext(inputPath), opts)
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

	outputPath := opts.Output
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		if cfg.Lossless {
			if mime == "image/jpeg" {
				ext = ".png"
			}
		}
		outputPath = defaultOutputPath(inputPath, ext)
	}

	if err := provider.Mark(inputPath, outputPath, payload, seed, cfg); err != nil {
		return nil, fmt.Errorf("embedding mark: %w", err)
	}

	return &MarkResult{
		Input:    inputPath,
		Output:   outputPath,
		Method:   provider.Method(),
		Payload:  payload.Hex(),
		Strength: cfg.Strength,
	}, nil
}
