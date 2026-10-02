package main

import (
	"bytes"
	"context"
	"os/exec"
)

// download returns the directory of one module, which the Go module proxy already holds or now fetches.
func download(ctx context.Context, e entry) (string, error) {
	data, stderr, err := modDownload(ctx, e)
	return moduleDirOf(e, data, stderr, err)
}

// modDownload asks the Go module proxy for one module, and it returns what the command wrote to stdout and to stderr.
func modDownload(ctx context.Context, e entry) ([]byte, string, error) {
	cmd := exec.CommandContext(ctx, "go", "mod", "download", "-json", e.path+"@"+e.version)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	return data, stderr.String(), err
}
