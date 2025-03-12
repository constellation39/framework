package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReadFileContent(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", absPath, err)
	}

	return string(content), nil
}
