package image

import (
	"fmt"
	"image"
	"math/bits"

	"github.com/a9sk/spymarks/internal/core"
)

// LSBProvider embeds marks in the least-significant bit of a selected channel.
type LSBProvider struct{}

// NewLSBProvider returns an LSB provider.
func NewLSBProvider() *LSBProvider {
	return &LSBProvider{}
}

func (p *LSBProvider) Method() string { return "lsb" }

func (p *LSBProvider) Supported(mime, ext string) bool {
	return mime == "image/png" || mime == "image/jpeg"
}

// MarkImage embeds payload directly into an RGBA image.
func (p *LSBProvider) MarkImage(img *image.RGBA, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}
	baseBits := ecc.Encode(payload.WithSyncBits(syncBits))

	bits := baseBits
	if cfg.Iterations > 1 {
		bits = make([]bool, 0, len(baseBits)*cfg.Iterations)
		for i := 0; i < cfg.Iterations; i++ {
			bits = append(bits, baseBits...)
		}
	}

	rng := core.NewDeterministicReader(seed)

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	capacity := w * h
	if capacity < len(bits) {
		return fmt.Errorf("image too small for payload: need %d pixels, have %d", len(bits), capacity)
	}

	idx := make([]int, capacity)
	for i := range idx {
		idx[i] = i
	}
	indices := rngPermute(rng, idx)

	ch := cfg.Channel
	if ch != 'r' && ch != 'g' && ch != 'b' && ch != 'a' {
		ch = 'g'
	}

	for i, bit := range bits {
		idx := indices[i]
		x, y := bounds.Min.X+idx%w, bounds.Min.Y+idx/w
		c := img.RGBAAt(x, y)
		var v uint8
		switch ch {
		case 'r':
			v = c.R
		case 'g':
			v = c.G
		case 'b':
			v = c.B
		case 'a':
			v = c.A
		default:
			v = c.G
		}
		if bit {
			v |= 1
		} else {
			v &^= 1
		}
		switch ch {
		case 'r':
			c.R = v
		case 'g':
			c.G = v
		case 'b':
			c.B = v
		case 'a':
			c.A = v
		default:
			c.G = v
		}
		img.SetRGBA(x, y, c)
	}
	return nil
}

func (p *LSBProvider) Mark(inputPath, outputPath string, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
	img, format, err := Load(inputPath)
	if err != nil {
		return err
	}

	if err := p.MarkImage(img, payload, seed, cfg); err != nil {
		return err
	}

	if cfg.DryRun {
		return nil
	}

	if cfg.Lossless {
		if format != "png" {
			format = "png"
		}
	}

	return SaveWithQuality(outputPath, img, format, cfg.Quality)
}

// CheckImage detects payload directly from an RGBA image.
func (p *LSBProvider) CheckImage(img *image.RGBA, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return nil, err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}

	rng := core.NewDeterministicReader(seed)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	capacity := w * h

	bitsToRead := int(float64(syncBits+payloadLen) * ecc.Expansion())
	if cfg.Iterations > 1 {
		bitsToRead *= cfg.Iterations
	}
	if capacity < bitsToRead {
		return nil, fmt.Errorf("image too small for payload")
	}

	idx := make([]int, capacity)
	for i := range idx {
		idx[i] = i
	}
	indices := rngPermute(rng, idx)

	ch := cfg.Channel
	if ch != 'r' && ch != 'g' && ch != 'b' && ch != 'a' {
		ch = 'g'
	}

	extracted := make([]bool, bitsToRead)
	for i := 0; i < bitsToRead; i++ {
		idx := indices[i]
		x, y := bounds.Min.X+idx%w, bounds.Min.Y+idx/w
		c := img.RGBAAt(x, y)
		var v uint8
		switch ch {
		case 'r':
			v = c.R
		case 'g':
			v = c.G
		case 'b':
			v = c.B
		case 'a':
			v = c.A
		default:
			v = c.G
		}
		extracted[i] = v&1 == 1
	}

	decoded := extracted
	if cfg.Iterations > 1 {
		decoded = majorityCombine(extracted, cfg.Iterations)
	}
	decoded = ecc.Decode(decoded)

	syncGot := core.PayloadFromBits(decoded[:syncBits]).Uint64()
	payloadBits := decoded[syncBits:]

	syncExpected := core.SyncPattern(syncBits) & ((1 << syncBits) - 1)
	matchingSync := syncBits - bits.OnesCount64(syncGot^syncExpected)
	confidence := float64(matchingSync) / float64(syncBits)

	if confidence < cfg.Threshold {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: confidence,
		}, nil
	}

	if cfg.ExtractOnly {
		return &core.CheckResult{
			Detected:   true,
			Method:     p.Method(),
			Payload:    core.PayloadFromBits(payloadBits).Hex(),
			Confidence: confidence,
		}, nil
	}

	payload := core.PayloadFromBits(payloadBits)
	return &core.CheckResult{
		Detected:   true,
		Method:     p.Method(),
		Payload:    payload.Hex(),
		Confidence: confidence,
	}, nil
}

func (p *LSBProvider) Check(inputPath string, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	img, _, err := Load(inputPath)
	if err != nil {
		return nil, err
	}
	return p.CheckImage(img, payloadLen, seed, cfg)
}

var _ image.Image = (*image.RGBA)(nil)
