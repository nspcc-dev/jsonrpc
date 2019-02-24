package codec

type (
	// Error is codec error representation
	Error struct {
		// A Number that indicates the error type that occurred.
		Code int `json:"code"` /* required */

		// A String providing a short description of the error.
		// The message SHOULD be limited to a concise single sentence.
		Message string `json:"message"` /* required */

		// A Primitive or Structured value that contains additional information about the error.
		Data interface{} `json:"data,omitempty"` /* optional */

		Internal error `json:"-"` // ignore
	}
)

const (
	// ErrServer Reserved for implementation-defined server-errors.
	ErrServer = -32000
	// ErrInvalidRequest The JSON sent is not a valid Request object.
	ErrInvalidRequest = -32600
	// ErrNoMethod The method does not exist / is not available.
	ErrNoMethod = -32601
	// ErrBadParams Invalid method parameter(s).
	ErrBadParams = -32602
	// ErrInternal Internal JSON-RPC error.
	ErrInternal = -32603
	// ErrParse Invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ErrParse = -32700
)

func (e *Error) Error() string { return e.Message }
