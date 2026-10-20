package text

import (
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/a9sk/spymarks/internal/core"
)

// SynonymProvider embeds marks by biasing synonym choice in English text.
// It is experimental and English-centric.
type SynonymProvider struct{}

// NewSynonymProvider returns a synonym-bias text provider.
func NewSynonymProvider() *SynonymProvider {
	return &SynonymProvider{}
}

func (p *SynonymProvider) Method() string { return "text" }

func (p *SynonymProvider) Supported(mime, ext string) bool {
	ext = strings.ToLower(ext)
	return mime == "text/plain" || ext == ".txt"
}

func (p *SynonymProvider) Mark(inputPath, outputPath string, payload *core.Payload, seed []byte, cfg *core.EmbedConfig) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading text: %w", err)
	}

	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}
	bits := ecc.Encode(payload.WithSyncBits(syncBits))

	text := string(data)
	tokens := tokenize(text)

	rng := core.NewDeterministicReader(seed)
	candidates := candidatePositions(tokens)
	if len(candidates) < len(bits) {
		return fmt.Errorf("text too short for payload: need %d synonym words, have %d", len(bits), len(candidates))
	}

	// Shuffle candidate positions deterministically.
	candidates = rngPermute(rng, candidates)[:len(bits)]

	for i, bit := range bits {
		pos := candidates[i]
		word := strings.ToLower(tokens[pos].value)
		syns := synonyms(word)
		if len(syns) == 0 {
			continue
		}
		// PRNG decides which synonym maps to 0 and which to 1.
		idx0, idx1 := rng.IntN(len(syns)), rng.IntN(len(syns))
		if idx0 == idx1 {
			idx1 = (idx1 + 1) % len(syns)
		}
		chosen := syns[idx1]
		if !bit {
			chosen = syns[idx0]
		}
		// Preserve capitalization roughly.
		tokens[pos].value = matchCase(tokens[pos].value, chosen)
	}

	output := detokenize(tokens)
	if cfg.DryRun {
		return nil
	}
	return os.WriteFile(outputPath, []byte(output), 0644)
}

func (p *SynonymProvider) Check(inputPath string, payloadLen int, seed []byte, cfg *core.EmbedConfig) (*core.CheckResult, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("reading text: %w", err)
	}

	ecc, err := core.GetECC(cfg.ECC)
	if err != nil {
		return nil, err
	}

	syncBits := cfg.SyncBits
	if syncBits <= 0 {
		syncBits = core.DefaultSyncBits
	}
	bitsToRead := int(float64(syncBits+payloadLen) * ecc.Expansion())

	text := string(data)
	tokens := tokenize(text)

	rng := core.NewDeterministicReader(seed)
	candidates := candidatePositions(tokens)
	if len(candidates) < bitsToRead {
		return nil, fmt.Errorf("text too short for payload")
	}
	candidates = rngPermute(rng, candidates)[:bitsToRead]

	extracted := make([]bool, bitsToRead)
	for i := 0; i < bitsToRead; i++ {
		pos := candidates[i]
		word := strings.ToLower(tokens[pos].value)
		syns := synonyms(word)
		if len(syns) == 0 {
			extracted[i] = false
			continue
		}
		idx0, idx1 := rng.IntN(len(syns)), rng.IntN(len(syns))
		if idx0 == idx1 {
			idx1 = (idx1 + 1) % len(syns)
		}
		extracted[i] = word == strings.ToLower(syns[idx1])
	}

	decoded := ecc.Decode(extracted)
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

	payload := core.PayloadFromBits(payloadBits)
	return &core.CheckResult{
		Detected:   true,
		Method:     p.Method(),
		Payload:    payload.Hex(),
		Confidence: confidence,
	}, nil
}

// token represents a token in the original text.
type token struct {
	value   string
	isWord  bool
	isSpace bool
}

var wordRE = regexp.MustCompile(`^[A-Za-z]+$`)

func tokenize(text string) []token {
	var tokens []token
	var current strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, token{value: current.String(), isWord: wordRE.MatchString(current.String())})
				current.Reset()
			}
			tokens = append(tokens, token{value: string(r), isSpace: true})
		} else if unicode.IsLetter(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, token{value: current.String(), isWord: wordRE.MatchString(current.String())})
				current.Reset()
			}
			tokens = append(tokens, token{value: string(r), isWord: false})
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, token{value: current.String(), isWord: wordRE.MatchString(current.String())})
	}
	return tokens
}

func detokenize(tokens []token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t.value)
	}
	return b.String()
}

func candidatePositions(tokens []token) []int {
	var out []int
	for i, t := range tokens {
		if t.isWord {
			w := strings.ToLower(t.value)
			if canonical(w) != "" {
				out = append(out, i)
			}
		}
	}
	return out
}

func canonical(word string) string {
	word = strings.ToLower(word)
	if _, ok := lexicon[word]; ok {
		return word
	}
	if c, ok := reverseLexicon[word]; ok {
		return c
	}
	return ""
}

func synonyms(word string) []string {
	c := canonical(word)
	if c == "" {
		return nil
	}
	return append([]string(nil), lexicon[c]...)
}

func rngPermute(rng *core.DeterministicReader, idx []int) []int {
	out := append([]int(nil), idx...)
	for i := len(out) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func matchCase(original, replacement string) string {
	if len(original) == 0 {
		return replacement
	}
	// All uppercase.
	allUpper := true
	for _, r := range original {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			allUpper = false
			break
		}
	}
	if allUpper {
		return strings.ToUpper(replacement)
	}
	// Capitalized.
	if unicode.IsUpper(rune(original[0])) {
		runes := []rune(replacement)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			return string(runes)
		}
	}
	return replacement
}

var lexicon = map[string][]string{
	"big":   {"large", "huge", "great"},
	"small": {"little", "tiny", "mini"},
	"fast":  {"quick", "rapid", "swift"},
	"slow":  {"sluggish", "gradual", "unhurried"},
	"happy": {"glad", "joyful", "cheerful"},
	"sad":   {"unhappy", "sorrowful", "gloomy"},
	"good":  {"fine", "excellent", "superior"},
	"bad":   {"poor", "inferior", "awful"},
	"start": {"begin", "commence", "initiate"},
	"end":   {"finish", "conclude", "terminate"},
	"help":  {"assist", "aid", "support"},
	"make":  {"create", "produce", "build"},
	"use":   {"utilize", "employ", "apply"},
	"show":  {"display", "exhibit", "demonstrate"},
	"tell":  {"inform", "notify", "relate"},
	"think": {"believe", "consider", "suppose"},
	"know":  {"understand", "recognize", "realize"},
	"want":  {"desire", "wish", "need"},
	"say":   {"state", "declare", "mention"},
	"go":    {"proceed", "move", "travel"},
	"come":  {"arrive", "approach", "reach"},
	"see":   {"observe", "notice", "view"},
	"look":  {"glance", "gaze", "stare"},
	"feel":  {"sense", "experience", "perceive"},
	"find":  {"discover", "locate", "detect"},
	"keep":  {"retain", "maintain", "preserve"},
	"give":  {"provide", "offer", "present"},
	"take":  {"seize", "grab", "obtain"},
	"work":  {"labor", "toil", "function"},
	"play":  {"perform", "act", "recreation"},
	"run":   {"sprint", "race", "dash"},
	"walk":  {"stroll", "stride", "wander"},
	"talk":  {"speak", "converse", "chat"},
	"read":  {"peruse", "scan", "study"},
	"write": {"compose", "draft", "record"},
	"eat":   {"consume", "devour", "dine"},
	"drink": {"sip", "gulp", "quaff"},
	"sleep": {"rest", "slumber", "doze"},
	"wake":  {"awaken", "rouse", "stir"},
}

var reverseLexicon = buildReverseLexicon()

func buildReverseLexicon() map[string]string {
	out := make(map[string]string)
	for word, syns := range lexicon {
		for _, s := range syns {
			out[strings.ToLower(s)] = word
		}
	}
	return out
}

var _ = filepath.Ext
