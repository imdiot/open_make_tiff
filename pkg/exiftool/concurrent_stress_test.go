package exiftool

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestConcurrentStress(t *testing.T) {
	exiftoolAvailable(t)

	e := newTestInstance(t)

	// Prepare 20 writable copies
	tmpDir := t.TempDir()
	files := make([]string, 20)
	for i := range files {
		data, err := os.ReadFile(testFile("ExifTool.jpg"))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(tmpDir, fmt.Sprintf("stress_%d.jpg", i))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = dst
	}

	// 100 concurrent mixed read/write operations
	const n = 100
	errCh := make(chan error, n)
	for i := range n {
		go func(idx int) {
			f := files[idx%len(files)]
			switch idx % 3 {
			case 0:
				_, err := e.ReadProperty(f, "Model")
				errCh <- err
			case 1:
				_, err := e.ReadMetadata(f)
				errCh <- err
			default:
				err := e.WriteMetadata(f, map[string]interface{}{
					"Comment": fmt.Sprintf("stress-%d", idx),
				})
				errCh <- err
			}
		}(i)
	}

	failCount := 0
	for i := range n {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent op %d error: %v", i, err)
			failCount++
		}
	}
	t.Logf("completed %d ops, %d failures", n, failCount)
}
