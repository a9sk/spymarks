# spymarks

> **research & educational poc**: a small go command-line tool that embeds and detects simple hidden marks ("spymarks") in media files.

**disclaimer:** this tool demonstrates how hidden tracking signals can be embedded in media files. Do not use it to track individuals without informed consent, to bypass platform policies, or for any unlawful purpose. The author does not endorse surveillance watermarking.

---

`spy` is a single binary with two subcommands:

```bash
spy mark  [flags] <file>   # embed a spymark
spy check [flags] <file>   # detect / verify a spymark
```

currently supported media:

- `image/png`
- `image/jpeg`
- `audio/wav` (FLAC decode/encode planned)
- `text/plain`

audio and text providers are experimental and English-centric respectively.

### global flags

| flag | description | default |
|------|-------------|---------|
| `--key` | secret key file or raw string | insecure demo key |
| `--method` | `dct`, `lsb`, `fft`, `text`, `auto` | `auto` |
| `--output` | output path for `mark` | `<input>-spymarked<ext>` |
| `--verbose` | print debug information | `false` |
| `--ecc` | error correction: `none`, `repeat`, `hamming` | `repeat` |
| `--only-images` | only consider image providers | `false` |
| `--only-audio` | only consider audio providers | `false` |
| `--only-text` | only consider text providers | `false` |
| `--block-size` | DCT block size | `8` |
| `--coeff-pair` | DCT coefficient pair `i,j` | `1,2` |
| `--channel` | channel: `y`, `r`, `g`, `b`, `a`, `cb`, `cr`, `gray` | `y` |
| `--quality` | JPEG quality (1-100) | `95` |
| `--lossless` | force PNG/FLAC lossless output | `false` |
| `--payload-bits` | payload length in bits | `64` |
| `--sync-bits` | sync preamble length in bits | `32` |
| `--iterations` | repeat payload embedding N times | `1` |

### `mark` flags

| flag | description | default |
|------|-------------|---------|
| `--message` | payload message | deterministic test pattern |
| `--strength` | embedding strength / alpha | `4.0` |
| `--raw-hex` | provide payload directly as hex | `""` |
| `--dry-run` | simulate embedding without writing | `false` |

### `check` flags

| flag | description | default |
|------|-------------|---------|
| `--strength` | strength used during marking | `4.0` |
| `--threshold` | detection threshold | `0.6` |
| `--raw-hex` | exected payload as hex | `""` |
| `--json` | output JSON | `false` |
| `--extract-only` | only print extracted payload | `false` |

---

## methods

### `dct` (default for images)

A patchwork-style DCT watermark over 8×8 luminance blocks. For each payload bit, two mid-frequency coefficients are nudged in opposite directions. The detector reads the signed difference. This method survives mild lossy re-compression (e.g., JPEG quality 90) much better than LSB.

### `lsb`

A fragile least-significant-bit watermark in the green channel. Fast and visually transparent, but easily destroyed by compression or cropping.

### `fft`

Experimental FFT patchwork over non-overlapping audio frames. Works with mono WAV files. Higher strength (e.g. `48-64`) is usually needed compared to images.

### `text`

Experimental synonym-bias watermarking for English text. Requires a large amount of text relative to the payload; use `--payload-bits`, `--sync-bits`, and `--ecc none` for shorter texts.
