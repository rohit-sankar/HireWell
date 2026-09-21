// Package storage abstracts "where uploaded files live" behind a small
// interface. LocalStorage (this file) is used in development. A production
// deployment swaps in an S3/GCS-backed implementation of the same
// FileStorage interface without touching any calling code -- that's the
// entire point of the abstraction.
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileStorage interface {
	// Save writes the given reader's contents under a safe, randomly
	// generated filename inside the given logical subdirectory (e.g.
	// "resumes"), returning the storage key (relative path) that can later
	// be passed to Open/Delete.
	Save(subdir, originalFilename string, r io.Reader) (storageKey string, err error)
	Open(storageKey string) (io.ReadCloser, error)
	Delete(storageKey string) error
	AbsolutePath(storageKey string) string
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

// safeFilename generates a random filename that preserves only the
// extension of the original upload -- this is the core defense against
// path traversal (no user-controlled path segments reach the filesystem)
// and against filename collisions/overwrites.
func safeFilename(originalFilename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(originalFilename))
	// allowlist of extensions we ever call Save() with for resumes; this is
	// belt-and-suspenders on top of the MIME/content validation done in
	// internal/resume before Save is ever called.
	switch ext {
	case ".pdf", ".doc", ".docx", ".txt":
	default:
		ext = ".bin"
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}

func (s *LocalStorage) Save(subdir, originalFilename string, r io.Reader) (string, error) {
	name, err := safeFilename(originalFilename)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(s.baseDir, filepath.Base(subdir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	storageKey := filepath.Join(filepath.Base(subdir), name)
	fullPath := filepath.Join(s.baseDir, storageKey)

	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		os.Remove(fullPath)
		return "", err
	}

	return storageKey, nil
}

// resolveSafe joins baseDir + storageKey and verifies the result is still
// inside baseDir, rejecting any "../" traversal attempts regardless of how
// storageKey was constructed upstream.
func (s *LocalStorage) resolveSafe(storageKey string) (string, error) {
	full := filepath.Join(s.baseDir, storageKey)
	absBase, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absFull, absBase+string(os.PathSeparator)) && absFull != absBase {
		return "", fmt.Errorf("invalid storage key: path traversal attempt")
	}
	return absFull, nil
}

func (s *LocalStorage) Open(storageKey string) (io.ReadCloser, error) {
	path, err := s.resolveSafe(storageKey)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *LocalStorage) Delete(storageKey string) error {
	path, err := s.resolveSafe(storageKey)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (s *LocalStorage) AbsolutePath(storageKey string) string {
	path, err := s.resolveSafe(storageKey)
	if err != nil {
		return ""
	}
	return path
}
