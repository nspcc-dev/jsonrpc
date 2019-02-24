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

	encoder         struct{}
	encoderSelector struct{}
)

var (
	DefaultEncoder         = &encoder{}
	DefaultEncoderSelector = &encoderSelector{}
)

func (_ *encoder) Encode(w http.ResponseWriter) io.Writer {
	return w
}

func (_ *encoderSelector) Select(_ *http.Request) Encoder {
	return DefaultEncoder
}
