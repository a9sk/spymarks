package core

// Options carries command-line configuration shared between mark and check.
type Options struct {
	Key         string
	Method      string
	Output      string
	Verbose     bool
	Message     string
	Strength    float64
	RawHex      string
	Threshold   float64
	ECC         string
	OnlyImages  bool
	OnlyAudio   bool
	OnlyText    bool
	BlockSize   int
	CoeffPair   string
	Channel     string
	Quality     int
	Lossless    bool
	PayloadBits int
	SyncBits    int
	Iterations  int
	DryRun      bool
	ExtractOnly bool
}

// MarkResult is the structured output of the mark command.
type MarkResult struct {
	Input    string
	Output   string
	Method   string
	Payload  string
	Strength float64
}

// CheckResult is the structured output of the check command.
type CheckResult struct {
	Detected   bool    `json:"detected"`
	Method     string  `json:"method"`
	Payload    string  `json:"payload"`
	Confidence float64 `json:"confidence"`
	Filename   string  `json:"filename"`
}

// EmbedConfig carries per-provider watermarking parameters.
type EmbedConfig struct {
	Strength    float64
	Threshold   float64
	ECC         string
	BlockSize   int
	CoeffPair   [2]int
	Channel     byte
	Quality     int
	Lossless    bool
	PayloadBits int
	SyncBits    int
	Iterations  int
	DryRun      bool
	ExtractOnly bool
}

// DefaultEmbedConfig returns an EmbedConfig with provider-agnostic defaults.
func DefaultEmbedConfig() *EmbedConfig {
	return &EmbedConfig{
		Strength:    4.0,
		Threshold:   0.6,
		ECC:         "repeat",
		BlockSize:   8,
		CoeffPair:   [2]int{1, 2},
		Channel:     'y',
		Quality:     95,
		Lossless:    false,
		PayloadBits: DefaultPayloadBits,
		SyncBits:    DefaultSyncBits,
		Iterations:  1,
	}
}

// Provider is the interface implemented by each media-specific embedder.
type Provider interface {
	// Mark embeds payload into input and writes to output.
	Mark(inputPath, outputPath string, payload *Payload, seed []byte, cfg *EmbedConfig) error
	// Check detects a payload in input and reports confidence.
	Check(inputPath string, payloadLen int, seed []byte, cfg *EmbedConfig) (*CheckResult, error)
	// Method returns the provider's canonical method name.
	Method() string
	// Supported returns whether this provider handles the given MIME/type hint.
	Supported(mime, ext string) bool
}
