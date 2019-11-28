package fs

import (
	"io"
	"os"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type noBomReader struct {
	io.Closer
	io.Reader
}

// NewFileReader creates io.ReadCloser that reads a file by ignoring BOM char
func NewFileReader(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
	unicodeReader := transform.NewReader(f, bom)
	r := &noBomReader{
		Closer: f,
		Reader: unicodeReader,
	}
	return r, nil
}
