package codec

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"unicode"
)

type (
	// CompressionSelector generates the compressed http encoder.
	CompressionSelector struct{}

	// gzipEncoder implements the gzip compressed http encoder.
	gzipEncoder struct{}

	gzipWriter struct {
		w *gzip.Writer
	}

	// flateEncoder implements the flate compressed http encoder.
	flateEncoder struct{}

	// flateWriter writes and closes the flate writer.
	flateWriter struct {
		w *flate.Writer
	}
)

func (gw *gzipWriter) Write(p []byte) (n int, err error) {
	defer gw.w.Close()
	return gw.w.Write(p)
}

func (enc *gzipEncoder) Encode(w http.ResponseWriter) io.Writer {
	w.Header().Set("Content-Encoding", "gzip")
	return &gzipWriter{gzip.NewWriter(w)}
}

func (fw *flateWriter) Write(p []byte) (n int, err error) {
	defer fw.w.Close()
	return fw.w.Write(p)
}

func (enc *flateEncoder) Encode(w http.ResponseWriter) io.Writer {
	fw, err := flate.NewWriter(w, flate.DefaultCompression)
	if err != nil {
		return w
	}
	w.Header().Set("Content-Encoding", "deflate")
	return &flateWriter{fw}
}

// acceptedEnc returns the first compression type in "Accept-Encoding" header
// field of the request.
func acceptedEnc(req *http.Request) string {
	encHeader := req.Header.Get("Accept-Encoding")
	if encHeader == "" {
		return ""
	}
	encTypes := strings.FieldsFunc(encHeader, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	for _, enc := range encTypes {
		if enc == "gzip" || enc == "deflate" {
			return enc
		}
	}
	return ""
}

// Select method selects the correct compression encoder based on http HEADER.
func (*CompressionSelector) Select(r *http.Request) Encoder {
	switch acceptedEnc(r) {
	case "gzip":
		return &gzipEncoder{}
	case "flate":
		return &flateEncoder{}
	}
	return DefaultEncoder
}
