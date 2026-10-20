// Package watermark exposes a small public API around the spymarks engine.
//
// It is intentionally thin, the cli should be the way of use.
package watermark

import (
	"github.com/a9sk/spymarks/internal"
	"github.com/a9sk/spymarks/internal/core"
)

// Config holds watermarking parameters.
type Config struct {
	Key         string
	Method      string
	Strength    float64
	Threshold   float64
	ECC         string
	BlockSize   int
	CoeffPair   string
	Channel     string
	Quality     int
	Lossless    bool
	PayloadBits int
	SyncBits    int
	Iterations  int
}

// Result is returned by Detect.
type Result struct {
	Detected   bool
	Method     string
	Payload    string
	Confidence float64
}

func defaultConfig(c *Config) {
	if c.Method == "" {
		c.Method = "auto"
	}
	if c.Strength == 0 {
		c.Strength = 4.0
	}
	if c.Threshold == 0 {
		c.Threshold = 0.6
	}
}

// Mark embeds a deterministic payload derived from message into inPath
// and writes the marked file to outPath.
func Mark(inPath, outPath, message string, cfg Config) (*core.MarkResult, error) {
	defaultConfig(&cfg)
	opts := &internal.Options{
		Key:         cfg.Key,
		Method:      cfg.Method,
		Output:      outPath,
		Message:     message,
		Strength:    cfg.Strength,
		ECC:         cfg.ECC,
		BlockSize:   cfg.BlockSize,
		CoeffPair:   cfg.CoeffPair,
		Channel:     cfg.Channel,
		Quality:     cfg.Quality,
		Lossless:    cfg.Lossless,
		PayloadBits: cfg.PayloadBits,
		SyncBits:    cfg.SyncBits,
		Iterations:  cfg.Iterations,
	}
	return internal.Mark(inPath, opts)
}

// Detect checks whether inPath contains a spymark generated from message.
func Detect(inPath, message string, cfg Config) (*Result, error) {
	defaultConfig(&cfg)
	opts := &internal.Options{
		Key:         cfg.Key,
		Method:      cfg.Method,
		Message:     message,
		Strength:    cfg.Strength,
		Threshold:   cfg.Threshold,
		ECC:         cfg.ECC,
		BlockSize:   cfg.BlockSize,
		CoeffPair:   cfg.CoeffPair,
		Channel:     cfg.Channel,
		Quality:     cfg.Quality,
		Lossless:    cfg.Lossless,
		PayloadBits: cfg.PayloadBits,
		SyncBits:    cfg.SyncBits,
		Iterations:  cfg.Iterations,
	}
	res, err := internal.Check(inPath, opts)
	if err != nil {
		return nil, err
	}
	return &Result{
		Detected:   res.Detected,
		Method:     res.Method,
		Payload:    res.Payload,
		Confidence: res.Confidence,
	}, nil
}
