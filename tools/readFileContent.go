package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReadFileContent(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v absolute path: %s", err, absPath)
	}

	return string(content), nil
}
