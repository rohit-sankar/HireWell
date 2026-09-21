package resume

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExtractText_TXT(t *testing.T) {
	text, err := ExtractText("text/plain", "resume.txt", []byte("Hello, this is my resume."))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello, this is my resume." {
		t.Errorf("expected passthrough text, got %q", text)
	}
}

// buildMinimalDocx constructs a real, valid .docx file in memory (a ZIP
// archive containing word/document.xml) so extractDOCX can be tested
// against an actual DOCX structure rather than a mock.
func buildMinimalDocx(t *testing.T, paragraphs []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	body.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, p := range paragraphs {
		body.WriteString(`<w:p><w:r><w:t>` + p + `</w:t></w:r></w:p>`)
	}
	body.WriteString(`</w:body></w:document>`)

	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := f.Write([]byte(body.String())); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractText_DOCX(t *testing.T) {
	data := buildMinimalDocx(t, []string{"John Doe", "Backend Developer", "Skills: Go, MySQL, Docker"})

	text, err := ExtractText("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "resume.docx", data)
	if err != nil {
		t.Fatalf("unexpected error extracting real docx: %v", err)
	}
	if !strings.Contains(text, "John Doe") {
		t.Errorf("expected extracted text to contain 'John Doe', got: %q", text)
	}
	if !strings.Contains(text, "Go, MySQL, Docker") {
		t.Errorf("expected extracted text to contain skills line, got: %q", text)
	}
}

func TestExtractText_DOCX_MissingDocumentXML(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("some_other_file.txt")
	f.Write([]byte("not a real docx"))
	zw.Close()

	_, err := ExtractText("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "fake.docx", buf.Bytes())
	if err == nil {
		t.Error("expected an error for a docx-like zip missing word/document.xml")
	}
}

func TestExtractText_LegacyDocRejectedHonestly(t *testing.T) {
	_, err := ExtractText("application/msword", "resume.doc", []byte{0xD0, 0xCF, 0x11, 0xE0})
	if err == nil {
		t.Error("legacy .doc must be explicitly rejected, not silently mis-parsed")
	}
	if !strings.Contains(err.Error(), "docx") {
		t.Errorf("error should guide the user to re-save as .docx/.pdf, got: %v", err)
	}
}

func TestExtractText_UnsupportedExtension(t *testing.T) {
	_, err := ExtractText("image/png", "resume.png", []byte{0x89, 0x50, 0x4E, 0x47})
	if err == nil {
		t.Error("expected an error for an unsupported file extension")
	}
}
