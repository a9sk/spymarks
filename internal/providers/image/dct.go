package image

import (
	"fmt"
	"image"
	"math"
	"math/bits"

	"github.com/a9sk/spymarks/internal/core"
)

const (
	defaultBlockSize = 8
	minVariance      = 4.0 // skip near-uniform blocks (std-dev < 2 on a 0-255 scale)
)

// DCTProvider embeds marks using block DCT coefficients.
type DCTProvider struct{}

// NewDCTProvider returns a DCT provider.
func NewDCTProvider() *DCTProvider {
	return &DCTProvider{}
}

func (p *DCTProvider) Method() string { return "dct" }

func (p *DCTProvider) Supported(mime, ext string) bool {
	return mime == "image/png" || mime == "image/jpeg"
}

// MarkImage embeds payload directly into an RGBA image.
func (p *DCTProvider) MarkImage(img *image.RGBA, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
	blockSize := cfg.BlockSize
	if blockSize <= 0 {
		blockSize = defaultBlockSize
	}

	plane := ToChannel(img, cfg.Channel)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

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

	bw, bh := w/blockSize, h/blockSize
	totalBlocks := bw * bh

	if totalBlocks < len(bits) {
		return fmt.Errorf("image too small for payload: need %d blocks, have %d", len(bits), totalBlocks)
	}

	indices := selectBlocks(plane, w, bw, bh, len(bits), rng)

	ciBase, cjBase := cfg.CoeffPair[0], cfg.CoeffPair[1]
	if ciBase < 0 || ciBase >= blockSize || cjBase < 0 || cjBase >= blockSize || ciBase == cjBase {
		return fmt.Errorf("invalid coefficient pair (%d,%d) for block size %d", ciBase, cjBase, blockSize)
	}

	for i, bit := range bits {
		idx := indices[i]
		bx, by := idx%bw, idx/bw
		block := extractBlock(plane, w, bx, by, blockSize)
		dct := dct2(block, blockSize)

		// Patchwork-style embedding on a pair of mid-frequency coefficients.
		// A secret sign flip swaps which coefficient is raised/lowered.
		ci, cj := ciBase, cjBase
		if rng.Bool() {
			ci, cj = cj, ci
		}
		sign := 1.0
		if rng.Bool() {
			sign = -1.0
		}

		// Forced-difference patchwork embedding. We set the signed
		// coefficient difference to a target magnitude so detection is
		// independent of the original block content.
		localVar := blockVariance(plane, w, bx, by, blockSize)
		norm := 1.0
		if localVar > minVariance {
			norm = math.Sqrt(localVar) / 8.0
			if norm < 1.0 {
				norm = 1.0
			}
		}

		target := cfg.Strength * norm
		if !bit {
			target = -target
		}
		diff := dct[ci][cj] - dct[cj][ci]
		adjust := sign*target - diff
		dct[ci][cj] += adjust / 2
		dct[cj][ci] -= adjust / 2

		block = idct2(dct, blockSize)
		placeBlock(plane, w, bx, by, blockSize, block)
	}

	FromChannel(img, cfg.Channel, plane)
	return nil
}

func (p *DCTProvider) Mark(inputPath, outputPath string, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
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
func (p *DCTProvider) CheckImage(img *image.RGBA, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	blockSize := cfg.BlockSize
	if blockSize <= 0 {
		blockSize = defaultBlockSize
	}

	plane := ToChannel(img, cfg.Channel)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return nil, err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}

	rng := core.NewDeterministicReader(seed)

	bw, bh := w/blockSize, h/blockSize
	totalBlocks := bw * bh

	bitsToRead := int(float64(syncBits+payloadLen) * ecc.Expansion())
	if cfg.Iterations > 1 {
		bitsToRead *= cfg.Iterations
	}
	if totalBlocks < bitsToRead {
		return nil, fmt.Errorf("image too small for payload")
	}

	indices := selectBlocks(plane, w, bw, bh, bitsToRead, rng)

	ciBase, cjBase := cfg.CoeffPair[0], cfg.CoeffPair[1]
	if ciBase < 0 || ciBase >= blockSize || cjBase < 0 || cjBase >= blockSize || ciBase == cjBase {
		return nil, fmt.Errorf("invalid coefficient pair (%d,%d) for block size %d", ciBase, cjBase, blockSize)
	}

	extracted := make([]bool, bitsToRead)
	for i := 0; i < bitsToRead; i++ {
		idx := indices[i]
		bx, by := idx%bw, idx/bw
		block := extractBlock(plane, w, bx, by, blockSize)
		dct := dct2(block, blockSize)

		ci, cj := ciBase, cjBase
		if rng.Bool() {
			ci, cj = cj, ci
		}
		sign := 1.0
		if rng.Bool() {
			sign = -1.0
		}
		extracted[i] = sign*(dct[ci][cj]-dct[cj][ci]) > 0
	}

	// If iterations were used, average the votes for each repeated bit stream.
	decoded := extracted
	if cfg.Iterations > 1 {
		decoded = majorityCombine(extracted, cfg.Iterations)
	}

	decoded = ecc.Decode(decoded)

	// First syncBits should match sync pattern.
	if len(decoded) < syncBits {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: 0.0,
		}, nil
	}

	syncGot := core.PayloadFromBits(decoded[:syncBits]).Uint64()
	payloadBits := decoded[syncBits:]

	syncExpected := core.SyncPattern(syncBits) & ((1 << syncBits) - 1)
	matchingSync := syncBits - bits.OnesCount64(syncGot^syncExpected)
	confidence := float64(matchingSync) / float64(syncBits)

	if matchingSync < syncBits/2 {
		return &core.CheckResult{
			Detected:   false,
			Method:     p.Method(),
			Payload:    "",
			Confidence: 0.0,
		}, nil
	}

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

func (p *DCTProvider) Check(inputPath string, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	img, _, err := Load(inputPath)
	if err != nil {
		return nil, err
	}
	return p.CheckImage(img, payloadLen, seed, cfg)
}

// majorityCombine averages bit votes across iterations. Each position in the
// output is the majority of the corresponding positions in each iteration.
func majorityCombine(bits []bool, iterations int) []bool {
	if iterations <= 1 || len(bits)%iterations != 0 {
		return append([]bool(nil), bits...)
	}
	unit := len(bits) / iterations
	out := make([]bool, unit)
	for i := 0; i < unit; i++ {
		ones := 0
		for j := 0; j < iterations; j++ {
			if bits[j*unit+i] {
				ones++
			}
		}
		out[i] = ones > iterations/2
	}
	return out
}

// selectBlocks returns a keyed permutation of blocks that have enough local
// variance to carry a watermark. Uniform blocks are skipped because the tiny
// AC coefficient tweaks get lost in rounding.
func selectBlocks(plane []float64, w, bw, bh, need int, rng *core.DeterministicReader) []int {
	blockSize := w / bw
	var candidates []int
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			if blockVariance(plane, w, bx, by, blockSize) >= minVariance {
				candidates = append(candidates, by*bw+bx)
			}
		}
	}
	if len(candidates) < need {
		// Fall back to all blocks if not enough textured ones.
		candidates = make([]int, bw*bh)
		for i := range candidates {
			candidates[i] = i
		}
	}
	return rngPermute(rng, candidates)[:need]
}

func blockVariance(plane []float64, w, bx, by, blockSize int) float64 {
	baseY := by * blockSize
	baseX := bx * blockSize
	var sum, sumSq float64
	for y := 0; y < blockSize; y++ {
		for x := 0; x < blockSize; x++ {
			v := plane[(baseY+y)*w+(baseX+x)]
			sum += v
			sumSq += v * v
		}
	}
	n := float64(blockSize * blockSize)
	mean := sum / n
	return sumSq/n - mean*mean
}

// rngPermute returns a Fisher-Yates shuffle of idx using rng.
func rngPermute(rng *core.DeterministicReader, idx []int) []int {
	out := append([]int(nil), idx...)
	for i := len(out) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func extractBlock(gray []float64, w, bx, by, blockSize int) [][]float64 {
	block := make([][]float64, blockSize)
	for i := range block {
		block[i] = make([]float64, blockSize)
	}
	baseY := by * blockSize
	baseX := bx * blockSize
	for y := 0; y < blockSize; y++ {
		for x := 0; x < blockSize; x++ {
			block[y][x] = gray[(baseY+y)*w+(baseX+x)]
		}
	}
	return block
}

func placeBlock(gray []float64, w, bx, by, blockSize int, block [][]float64) {
	baseY := by * blockSize
	baseX := bx * blockSize
	for y := 0; y < blockSize; y++ {
		for x := 0; x < blockSize; x++ {
			v := block[y][x]
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			gray[(baseY+y)*w+(baseX+x)] = v
		}
	}
}

// dct2 computes a type-II 2D DCT on an NxN block.
func dct2(block [][]float64, n int) [][]float64 {
	tmp := make([][]float64, n)
	for i := range tmp {
		tmp[i] = make([]float64, n)
	}
	for y := 0; y < n; y++ {
		for u := 0; u < n; u++ {
			sum := 0.0
			for x := 0; x < n; x++ {
				sum += block[y][x] * math.Cos(math.Pi*float64(u)*(float64(x)+0.5)/float64(n))
			}
			cu := 1.0
			if u == 0 {
				cu = 1.0 / math.Sqrt2
			}
			tmp[y][u] = sum * cu * math.Sqrt(2.0/float64(n))
		}
	}
	out := make([][]float64, n)
	for i := range out {
		out[i] = make([]float64, n)
	}
	for x := 0; x < n; x++ {
		for v := 0; v < n; v++ {
			sum := 0.0
			for y := 0; y < n; y++ {
				sum += tmp[y][x] * math.Cos(math.Pi*float64(v)*(float64(y)+0.5)/float64(n))
			}
			cv := 1.0
			if v == 0 {
				cv = 1.0 / math.Sqrt2
			}
			out[v][x] = sum * cv * math.Sqrt(2.0/float64(n))
		}
	}
	return out
}

// idct2 computes a type-III 2D inverse DCT on an NxN block.
func idct2(dct [][]float64, n int) [][]float64 {
	tmp := make([][]float64, n)
	for i := range tmp {
		tmp[i] = make([]float64, n)
	}
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			sum := 0.0
			for v := 0; v < n; v++ {
				cv := 1.0
				if v == 0 {
					cv = 1.0 / math.Sqrt2
				}
				sum += cv * dct[v][x] * math.Cos(math.Pi*float64(v)*(float64(y)+0.5)/float64(n))
			}
			tmp[y][x] = sum * math.Sqrt(2.0/float64(n))
		}
	}
	out := make([][]float64, n)
	for i := range out {
		out[i] = make([]float64, n)
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			sum := 0.0
			for u := 0; u < n; u++ {
				cu := 1.0
				if u == 0 {
					cu = 1.0 / math.Sqrt2
				}
				sum += cu * tmp[y][u] * math.Cos(math.Pi*float64(u)*(float64(x)+0.5)/float64(n))
			}
			out[y][x] = sum * math.Sqrt(2.0/float64(n))
		}
	}
	return out
}

// ensure image import is used.
var _ image.Image = (*image.RGBA)(nil)
