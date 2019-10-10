package codec

import (
	"io"
	"net/http"
)

type (
	// Encoder interface contains the encoder for http response.
	// Eg. gzip, flate compressions.
	Encoder interface {
		Encode(w http.ResponseWriter) io.Writer
	}

	// EncoderSelector interface provides a way to select encoder using the http
	// request. Typically people can use this to check HEADER of the request and
	// figure out client capabilities.
	// Eg. "Accept-Encoding" tells about supported compressions.
	EncoderSelector interface {
		Select(r *http.Request) Encoder
	}

	encResponseWriter struct {
		http.ResponseWriter
		enc Encoder
	}

	encoder         int
	encoderSelector int
)

var (
	// DefaultEncoder for request
	DefaultEncoder = encoder(0)
	// DefaultEncoderSelector for request
	DefaultEncoderSelector = encoderSelector(0)
)

func (encoder) Encode(w http.ResponseWriter) io.Writer { return w }

func (encoderSelector) Select(_ *http.Request) Encoder { return DefaultEncoder }

func (w *encResponseWriter) Write(data []byte) (int, error) {
	return w.enc.Encode(w.ResponseWriter).Write(data)
}

func NewEncodedResponse(w http.ResponseWriter, enc Encoder) http.ResponseWriter {
	return &encResponseWriter{
		ResponseWriter: w,
		enc:            enc,
	}
}
