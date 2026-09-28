package idiom

import (
	"errors"
	"io"
	"os"
)

var errEmpty = errors.New("empty")

func readAll(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errEmpty
	}
	return data, nil
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func writeAll(path string, parts [][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if _, err := f.Write(p); err != nil {
			return err
		}
	}
	return f.Close()
}
