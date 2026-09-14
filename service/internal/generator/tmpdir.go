package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TemporaryOutputDir is the per-build working directory ({outputDir}_tmp) under baseOutputDir.
func TemporaryOutputDir(baseOutputDir, outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("output dir is empty")
	}

	base, err := filepath.Abs(baseOutputDir)
	if err != nil {
		return "", err
	}

	tmp, err := filepath.Abs(filepath.Join(base, outputDir+"_tmp"))
	if err != nil {
		return "", err
	}

	if !isUnderDir(base, tmp) {
		return "", fmt.Errorf("cache path %s is outside output directory", tmp)
	}

	return tmp, nil
}

// ClearTemporaryOutputDir removes the build temp directory for outputDir.
func ClearTemporaryOutputDir(baseOutputDir, outputDir string) (string, error) {
	tmp, err := TemporaryOutputDir(baseOutputDir, outputDir)
	if err != nil {
		return tmp, err
	}
	return tmp, os.RemoveAll(tmp)
}

func isUnderDir(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
