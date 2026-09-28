package cas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	envdriver "github.com/lewtec/modot/internal/driver/env"
)

type CASWriter struct {
	tempFile *os.File
	writer   io.Writer
	hasher   io.Writer
	dir      string
	ctx      context.Context
}

func NewCASWriter(ctx context.Context) (*CASWriter, error) {
	dataDir, err := envdriver.GetUserDataDir(ctx)
	if err != nil {
		return nil, err
	}
	genDir := filepath.Join(dataDir, "generated")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return nil, err
	}

	tempFile, err := os.CreateTemp(genDir, ".tmp-*")
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)

	return &CASWriter{
		tempFile: tempFile,
		writer:   writer,
		hasher:   hasher,
		dir:      genDir,
		ctx:      ctx,
	}, nil
}

func (c *CASWriter) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

func (c *CASWriter) Seal() (string, error) {
	if err := c.tempFile.Close(); err != nil {
		return "", err
	}
	hash := c.hasher.(interface{ Sum(b []byte) []byte }).Sum(nil)
	hashStr := hex.EncodeToString(hash)
	finalPath := filepath.Join(c.dir, hashStr)

	if _, err := os.Stat(finalPath); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(c.tempFile.Name(), finalPath); err != nil {
			return "", err
		}
	} else {
		if err := func() error { return os.Remove(c.tempFile.Name()) }(); err != nil {
			slog.ErrorContext(c.ctx, "unexpected error", "op", "remove", "error", err)
		}
	}

	return finalPath, nil
}
