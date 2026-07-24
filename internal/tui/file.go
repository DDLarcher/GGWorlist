package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readFileImpl reads a file with path traversal and size validation.
// Rejects paths containing "..", directories, and files larger than 100 MB.
func readFileImpl(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return nil, fmt.Errorf("path contains '..' (path traversal not allowed)")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}
	if info.Size() > 100*1000*1000 {
		return nil, fmt.Errorf("file is too large (max 100 MB)")
	}
	return os.ReadFile(clean)
}