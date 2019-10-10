package misc

import (
	"fmt"
	"net/http"
)

// HTTPError represents an error that occurred while handling a request.
type HTTPError struct {
	Code    int
	Message interface{}
}

const (
	charsetUTF8 = "charset=UTF-8"

	// HeaderContentType constant
	HeaderContentType = "Content-Type"
	// HeaderAcceptEncoding constant
	HeaderAcceptEncoding = "Accept-Encoding"
	// HeaderContentEncoding constant
	HeaderContentEncoding = "Content-Encoding"
	// MIMEApplicationJSON constant
	MIMEApplicationJSON = "application/json"
	// MIMEApplicationJSONCharsetUTF8 constant
	MIMEApplicationJSONCharsetUTF8 = MIMEApplicationJSON + "; " + charsetUTF8
	// HeaderXContentTypeOptions constant
	HeaderXContentTypeOptions = "X-Content-Type-Options"
)

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
}
