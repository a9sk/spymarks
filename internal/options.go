package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a9sk/spymarks/internal/core"
)

// Options carries command-line configuration shared between mark and check.
type Options = core.Options

// MarkResult is the structured output of the mark command.
type MarkResult = core.MarkResult

// CheckResult is the structured output of the check command.
type CheckResult = core.CheckResult

// Provider is the interface implemented by each media-specific embedder.
type Provider = core.Provider

// EmbedConfig is the alias for the per-provider config struct.
type EmbedConfig = core.EmbedConfig

// ToEmbedConfig converts Options into an EmbedConfig, applying defaults.
func ToEmbedConfig(opts *Options) (*EmbedConfig, error) {
	cfg := core.DefaultEmbedConfig()

	if opts.Strength != 0 {
		cfg.Strength = opts.Strength
	}
	if opts.Threshold != 0 {
		cfg.Threshold = opts.Threshold
	}
	if opts.ECC != "" {
		if _, err := core.GetECC(opts.ECC); err != nil {
			return nil, err
		}
		cfg.ECC = opts.ECC
	}
	if opts.BlockSize != 0 {
		cfg.BlockSize = opts.BlockSize
	}
	if opts.CoeffPair != "" {
		parts := strings.Split(opts.CoeffPair, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("--coeff-pair must be of form i,j")
		}
		i, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("--coeff-pair: %w", err)
		}
		j, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("--coeff-pair: %w", err)
		}
		cfg.CoeffPair = [2]int{i, j}
	}
	if opts.Channel != "" {
		ch := strings.ToLower(opts.Channel)
		switch ch {
		case "r", "g", "b", "a", "y":
			cfg.Channel = ch[0]
		case "cb":
			cfg.Channel = 'c'
		case "cr":
			cfg.Channel = 'C'
		case "gray", "grey":
			cfg.Channel = 'y'
		default:
			return nil, fmt.Errorf("unsupported channel: %s", opts.Channel)
		}
	}
	if opts.Quality != 0 {
		cfg.Quality = opts.Quality
	}
	if opts.Lossless {
		cfg.Lossless = true
	}
	if opts.PayloadBits != 0 {
		cfg.PayloadBits = opts.PayloadBits
	}
	if opts.SyncBits != 0 {
		cfg.SyncBits = opts.SyncBits
	}
	if opts.Iterations != 0 {
		cfg.Iterations = opts.Iterations
	}
	if opts.DryRun {
		cfg.DryRun = true
	}
	if opts.ExtractOnly {
		cfg.ExtractOnly = true
	}
	return cfg, nil
}
