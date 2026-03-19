package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/richardnascimento18/devdock/internal/fileutil"
)

func indexPath(configDir string) string {
	return filepath.Join(configDir, "index.json")
}

func SaveIndex(configDir string, p []Project) {
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = fileutil.WriteFileAtomic(indexPath(configDir), data, 0o600)
}

func LoadIndex(configDir string) ([]Project, error) {
	var p []Project
	data, err := os.ReadFile(indexPath(configDir))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return p, nil
}
