package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type TempDir struct {
	path string
}

func NewTempDir(prefix string) (*TempDir, error) {
	u := uuid.New()
	path := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s", prefix, u))
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return &TempDir{path: path}, nil
}

func (t *TempDir) Path() string { return t.path }

func (t *TempDir) Cleanup() error { return os.RemoveAll(t.path) }
