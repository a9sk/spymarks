// spy is a research and educational proof-of-concept tool for embedding and
// detecting simple hidden marks ("spymarks") in media files.
//
// This tool is intentionally simple and is not a robust industrial watermarking
// system.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/a9sk/spymarks/internal"
)

const (
	exitDetected = 0
	exitNone     = 1
	exitError    = 2
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitError)
	}

	subcommand := os.Args[1]
	if subcommand == "-h" || subcommand == "--help" {
		printUsage()
		os.Exit(0)
	}
	if subcommand == "-v" || subcommand == "--version" || subcommand == "version" {
		fmt.Printf("spy version %s (%s)\n", version, commit)
		os.Exit(0)
	}
	if subcommand == "-l" || subcommand == "--list-methods" || subcommand == "list-methods" {
		fmt.Println("Supported methods:")
		for _, m := range internal.ListMethods() {
			fmt.Printf("  - %s\n", m)
		}
		os.Exit(0)
	}

	if subcommand != "mark" && subcommand != "check" {
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n", subcommand)
		printUsage()
		os.Exit(exitError)
	}

	fs := flag.NewFlagSet(subcommand, flag.ExitOnError)

	var (
		key     string
		method  string
		output  string
		verbose bool

		message  string
		strength float64
		rawHex   string

		threshold  float64
		jsonOutput bool

		ecc         string
		onlyImages  bool
		onlyAudio   bool
		onlyText    bool
		blockSize   int
		coeffPair   string
		channel     string
		quality     int
		lossless    bool
		payloadBits int
		syncBits    int
		iterations  int
		dryRun      bool
		extractOnly bool
	)

	fs.StringVar(&key, "key", "", "secret key file or raw string (default: insecure demo key)")
	fs.StringVar(&method, "method", "auto", "embedding method: dct, lsb, fft, text, auto")
	fs.StringVar(&output, "output", "", "output path for mark (default: <input>-spymarked<ext>)")
	fs.BoolVar(&verbose, "verbose", false, "print debug information")

	fs.StringVar(&ecc, "ecc", "repeat", "error correction: none, repeat, hamming")
	fs.BoolVar(&onlyImages, "only-images", false, "only consider image providers")
	fs.BoolVar(&onlyAudio, "only-audio", false, "only consider audio providers")
	fs.BoolVar(&onlyText, "only-text", false, "only consider text providers")
	fs.IntVar(&blockSize, "block-size", 8, "DCT block size")
	fs.StringVar(&coeffPair, "coeff-pair", "1,2", "DCT coefficient pair i,j")
	fs.StringVar(&channel, "channel", "y", "channel: y, r, g, b, a, cb, cr, gray")
	fs.IntVar(&quality, "quality", 95, "JPEG quality (1-100)")
	fs.BoolVar(&lossless, "lossless", false, "force PNG/FLAC lossless output")
	fs.IntVar(&payloadBits, "payload-bits", 64, "payload length in bits")
	fs.IntVar(&syncBits, "sync-bits", 32, "sync preamble length in bits")
	fs.IntVar(&iterations, "iterations", 1, "repeat payload embedding N times")

	if subcommand == "mark" {
		fs.StringVar(&message, "message", "", "payload message (default: deterministic test pattern)")
		fs.Float64Var(&strength, "strength", 4.0, "embedding strength / alpha")
		fs.StringVar(&rawHex, "raw-hex", "", "provide payload directly as hex")
		fs.BoolVar(&dryRun, "dry-run", false, "simulate embedding without writing")
	} else {
		fs.Float64Var(&strength, "strength", 4.0, "embedding strength used during marking")
		fs.StringVar(&rawHex, "raw-hex", "", "expected payload as hex")
		fs.Float64Var(&threshold, "threshold", 0.6, "detection threshold")
		fs.BoolVar(&jsonOutput, "json", false, "output results as JSON")
		fs.BoolVar(&extractOnly, "extract-only", false, "only print extracted payload")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		os.Exit(exitError)
	}

	args := fs.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: spy %s [flags] <file>\n", subcommand)
		fs.PrintDefaults()
		os.Exit(exitError)
	}
	path := args[0]

	opts := &internal.Options{
		Key:         key,
		Method:      method,
		Output:      output,
		Verbose:     verbose,
		Message:     message,
		Strength:    strength,
		RawHex:      rawHex,
		Threshold:   threshold,
		ECC:         ecc,
		OnlyImages:  onlyImages,
		OnlyAudio:   onlyAudio,
		OnlyText:    onlyText,
		BlockSize:   blockSize,
		CoeffPair:   coeffPair,
		Channel:     channel,
		Quality:     quality,
		Lossless:    lossless,
		PayloadBits: payloadBits,
		SyncBits:    syncBits,
		Iterations:  iterations,
		DryRun:      dryRun,
		ExtractOnly: extractOnly,
	}

	switch subcommand {
	case "mark":
		res, err := internal.Mark(path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(exitError)
		}
		fmt.Printf("Input:    %s\n", res.Input)
		fmt.Printf("Output:   %s\n", res.Output)
		fmt.Printf("Method:   %s\n", res.Method)
		fmt.Printf("Payload:  %s\n", res.Payload)
		fmt.Printf("Strength: %.2f\n", res.Strength)

	case "check":
		res, err := internal.Check(path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(exitError)
		}
		if extractOnly {
			fmt.Println(res.Payload)
		} else if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(res); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding json: %v\n", err)
				os.Exit(exitError)
			}
		} else {
			fmt.Printf("Detected:   %v\n", res.Detected)
			fmt.Printf("Method:     %s\n", res.Method)
			fmt.Printf("Payload:    %s\n", res.Payload)
			fmt.Printf("Confidence: %.2f\n", res.Confidence)
		}
		if res.Detected {
			os.Exit(exitDetected)
		}
		os.Exit(exitNone)
	}
}

func printUsage() {
	fmt.Println(`spy - embed and detect hidden marks in media files (research PoC)

Usage:
  spy mark  [flags] <file>
  spy check [flags] <file>

Global flags:
  --key string       Secret key file or raw string (default insecure demo key)
  --method string    Embedding method: dct, lsb, fft, text, auto (default auto)
  --output string    Output path for mark (default <input>-spymarked<ext>)
  --verbose          Print debug information
  --version          Print version and exit
  --list-methods     Print supported methods and exit

Mark flags:
  --message string   Payload message (default deterministic test pattern)
  --strength float   Embedding strength / alpha (default 4.0)
  --raw-hex string   Provide payload directly as hex

Check flags:
  --strength float   Embedding strength used during marking (default 4.0)
  --raw-hex string   Expected payload as hex
  --threshold float  Detection threshold (default 0.6)
  --json             Output results as JSON

DISCLAIMER: This tool is a research and educational proof of concept. Do not use
it to track individuals without informed consent, to bypass platform policies,
or for any unlawful purpose.`)
}
