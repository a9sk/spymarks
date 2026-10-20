#!/usr/bin/env bash
# Example commands for the spy proof-of-concept.
# Run from the repository root after building: go build ./cmd/spy

set -e

cd "$(dirname "$0")/.."

# Build the binary.
go build ./cmd/spy

# Create a sample image if none exists.
if [ ! -f testdata/sample.png ]; then
  mkdir -p testdata
  MAGICK="${MAGICK:-convert}"
  $MAGICK -size 256x256 xc:lightblue -pointsize 30 -fill black -gravity center -annotate +0+0 "Spymarks\nTest" testdata/sample.png
fi

# Embed a DCT spymark.
./spy mark --key my.key --message "user-1234" testdata/sample.png

# Detect it.
./spy check --key my.key testdata/sample-spymarked.png

# JSON output.
./spy check --key my.key --json testdata/sample-spymarked.png

# Fragile LSB method.
./spy mark --method lsb --key my.key --message "user-1234" testdata/sample.png
./spy check --method lsb --key my.key testdata/sample-spymarked.png

# Show that the original image is not detected.
./spy check --key my.key testdata/sample.png || true
