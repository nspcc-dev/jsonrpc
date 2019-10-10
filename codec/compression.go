package codec

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/nspcc-dev/jsonrpc/misc"
)

type (
	// CompressionSelector generates the compressed http encoder.
	CompressionSelector struct{}

	// gzipEncoder implements the gzip compressed http encoder.
	gzipEncoder struct{}

	gzipWriter struct {
		w *gzip.Writer
	}

	// deflateEncoder implements the deflate compressed http encoder.
	deflateEncoder struct{}

	// deflateWriter writes and closes the deflate writer.
	deflateWriter struct {
		w *flate.Writer
	}
)

func (gw *gzipWriter) Write(p []byte) (n int, err error) {
	defer func() {
		if err == nil {
			err = gw.w.Close()
		}
	}()
	return gw.w.Write(p)
}

func (enc *gzipEncoder) Encode(w http.ResponseWriter) io.Writer {
	w.Header().Set(misc.HeaderContentEncoding, "gzip")
	return &gzipWriter{gzip.NewWriter(w)}
}

func (fw *deflateWriter) Write(p []byte) (n int, err error) {
	defer func() {
		if err == nil {
			err = fw.w.Close()
		}
	}()
	return fw.w.Write(p)
}

func (enc *deflateEncoder) Encode(w http.ResponseWriter) io.Writer {
	// we use default compression level, error can be omitted
	fw, _ := flate.NewWriter(w, flate.DefaultCompression)
	w.Header().Set(misc.HeaderContentEncoding, "deflate")
	return &deflateWriter{fw}
}

// acceptedEnc returns the first compression type in "Accept-Encoding" header
// field of the request.
func acceptedEnc(req *http.Request) string {
	encHeader := req.Header.Get(misc.HeaderAcceptEncoding)
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
	switch enc := acceptedEnc(r); enc {
	case "gzip":
		return &gzipEncoder{}
	case "deflate":
		return &deflateEncoder{}
	default:
		return DefaultEncoder
	}
}
