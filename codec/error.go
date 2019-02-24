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
	ErrServer         = -32000
	ErrInvalidRequest = -32600
	ErrNoMethod       = -32601
	ErrBadParams      = -32602
	ErrInternal       = -32603
	ErrParse          = -32700
)

func (e *Error) Error() string { return e.Message }
