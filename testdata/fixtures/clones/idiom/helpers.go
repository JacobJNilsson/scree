package idiom

import (
	"io"
	"os"
)

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func removeAll(paths []string) error {
	for _, p := range paths {
		if err := os.Remove(p); err != nil {
			return err
		}
	}
	return nil
}
