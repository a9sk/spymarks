package internal

import (
	"os"
	"path/filepath"
	"strings"
)

// readFileHeader reads the first n bytes of path.
func readFileHeader(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	m, err := f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:m], nil
}

// defaultOutputPath returns <input>-spymarked<ext> for a mark operation.
func defaultOutputPath(input, ext string) string {
	base := strings.TrimSuffix(input, filepath.Ext(input))
	return base + "-spymarked" + ext
}
