package resume

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const (
	MaxResumeFileSize = 10 * 1024 * 1024 // 10MB hard ceiling regardless of config, belt-and-suspenders
)

var allowedExtensions = map[string]bool{".pdf": true, ".doc": true, ".docx": true, ".txt": true}

// magic byte signatures for the formats we accept. Trusting only the
// filename extension / client-provided Content-Type is not sufficient --
// both are attacker-controlled. Sniffing the actual file header is what
// makes this a real validation rather than a cosmetic check.
var magicSignatures = map[string][]byte{
	".pdf":  []byte("%PDF"),
	".docx": {0x50, 0x4B, 0x03, 0x04}, // docx is a zip archive (PK\x03\x04)
	".doc":  {0xD0, 0xCF, 0x11, 0xE0}, // OLE compound file header
}

// ValidateUpload checks extension, declared size, and (for pdf/docx/doc)
// magic bytes. .txt has no reliable magic number, so we only bound its size
// and extension here.
func ValidateUpload(fh *multipart.FileHeader, maxSize int64) error {
	if maxSize <= 0 || maxSize > MaxResumeFileSize {
		maxSize = MaxResumeFileSize
	}
	if fh.Size <= 0 {
		return fmt.Errorf("uploaded file is empty")
	}
	if fh.Size > maxSize {
		return fmt.Errorf("file exceeds maximum size of %d bytes", maxSize)
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("unsupported file type %q: allowed types are .pdf, .doc, .docx, .txt", ext)
	}

	if sig, ok := magicSignatures[ext]; ok {
		f, err := fh.Open()
		if err != nil {
			return fmt.Errorf("could not open uploaded file: %w", err)
		}
		defer f.Close()

		header := make([]byte, len(sig))
		n, _ := f.Read(header)
		if n < len(sig) || !bytes.Equal(header[:n], sig) {
			return fmt.Errorf("file content does not match its extension (%s) -- possible corrupted or disguised file", ext)
		}
	}

	return nil
}
