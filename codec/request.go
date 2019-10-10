package codec

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nspcc-dev/jsonrpc/misc"
)

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

type (
	// serverRequest represents a JSON-RPC request received by the server.
	serverRequest struct {
		// JSON-RPC protocol.
		Version json.Number `json:"jsonrpc,omitempty"`

		// The request id. MUST be a string, number or null.
		// Our implementation will not do type checking for id.
		// It will be copied as it is.
		ID *json.Number `json:"id,int,omitempty"`

		// A String containing the name of the method to be invoked.
		Method string `json:"method,omitempty"`

		// A Structured value to pass as arguments to the method.
		Params json.RawMessage `json:"params,omitempty"`
	}

	// serverResponse represents a JSON-RPC response returned by the server.
	serverResponse struct {
		// JSON-RPC protocol.
		Version json.Number `json:"jsonrpc,string"`

		// This must be the same id as the request it is responding to.
		ID *json.Number `json:"id,int"`

		// The Object that was returned by the invoked method. This must be null
		// in case there was an error invoking the method.
		// As per spec the member will be omitted if there was an error.
		Result interface{} `json:"result,omitempty"`

		// An Error object if there was an error invoking the method. It must be
		// null if there was no error.
		// As per spec the member will be omitted if there was no error.
		Error *Error `json:"error,omitempty"`
	}

	// Interface codec creates a CodecRequest to process each request.
	Interface interface {
		NewRequest(http.ResponseWriter, *http.Request) (Request, error)
	}

	// Request decodes a request and encodes a response using a specific
	// serialization scheme.
	Request interface {
		// HandleError from input and request instance
		HandleError(err error) bool
		// Reads the request and returns the RPC method name.
		Method() string
		// Reads the request filling the RPC method args.
		ReadRequest(interface{}) error
		// Writes the response using the RPC method reply.
		WriteResponse(interface{})
		// Writes an error produced by the server.
		WriteError(status int, err error)
	}

	// codec creates a Request to process each request.
	codec struct {
		encSel EncoderSelector
	}

	// request decodes and encodes a single request.
	request struct {
		writer  http.ResponseWriter
		request *serverRequest
		encoder Encoder
	}
)

// Version of json-rpc protocol
const Version = "2.0"

// NewCustom returns a new JSON codec based on passed encoder selector.
func NewCustom(encSel EncoderSelector) Interface {
	return &codec{encSel: encSel}
}

// NewCodec returns a new JSON codec.
func NewCodec() Interface {
	return NewCustom(DefaultEncoderSelector)
}

// NewRequest returns a Request.
func (c *codec) NewRequest(w http.ResponseWriter, r *http.Request) (Request, error) {
	return newCodecRequest(w, r, c.encSel.Select(r))
}

// newCodecRequest returns a new Request.
func newCodecRequest(w http.ResponseWriter, r *http.Request, encoder Encoder) (Request, error) {
	var (
		req = new(serverRequest)
		err error
	)

	defer func() {
		if r.Body != nil {
			_ = r.Body.Close()
		}
	}()

	if r.Method != http.MethodPost {
		return nil, &Error{
			Code:    ErrInvalidRequest,
			Message: "rpc: POST method required, received " + r.Method,
		}
		// return &request{request: req, err: err, encoder: encoder}
	} else if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Decode the request body and check if RPC method is valid.
		return nil, &Error{
			Code:     ErrParse,
			Message:  err.Error(),
			Data:     req,
			Internal: err,
		}
	} else if req.Version != Version {
		return nil, &Error{
			Code:    ErrInvalidRequest,
			Message: "jsonrpc must be " + Version,
			Data:    req,
		}
	}
	return &request{writer: w, request: req, encoder: encoder}, nil
}

func (c *request) HandleError(err error) bool {
	switch err := err.(type) {
	case nil:
		return false
	case *misc.HTTPError:
		c.WriteError(err.Code, err)
	case *Error:
		switch err.Internal.(type) {
		case *json.SyntaxError, *json.UnmarshalTypeError:
			err.Message = "cannot unmarshal request"
		}
		c.WriteError(err.Code, err)
	default:
		c.WriteError(http.StatusBadRequest, err)
	}

	return true
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *request) Method() string {
	return c.request.Method
}

// ReadRequest fills the request object for the RPC method.
//
// ReadRequest parses request parameters in two supported forms in
// accordance with http://www.jsonrpc.org/specification#parameter_structures
//
// by-position: params MUST be an Array, containing the
// values in the Server expected order.
//
// by-name: params MUST be an Object, with member names
// that match the Server expected parameter names. The
// absence of expected names MAY result in an error being
// generated. The names MUST match exactly, including
// case, to the method's expected parameters.
func (c *request) ReadRequest(args interface{}) error {
	if c.request.Params != nil {
		// Note: if c.request.Params is nil it's not an error, it's an optional member.
		// JSON params structured object. Unmarshal to the args object.
		if err := json.Unmarshal(c.request.Params, args); err != nil {
			// Clearly JSON params is not a structured object,
			// fallback and attempt an unmarshal with JSON params as
			// array value and RPC params is struct. Unmarshal into
			// array containing the request struct.
			if err = json.Unmarshal(c.request.Params, &args); err != nil {
				return &Error{
					Code:     ErrBadParams,
					Message:  err.Error(),
					Data:     c.request.Params,
					Internal: err,
				}
			}
		}
	}
	return nil
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
func (c *request) WriteResponse(reply interface{}) {
	res := &serverResponse{
		Version: Version,
		Result:  reply,
		ID:      c.request.ID,
	}
	c.writeServerResponse(res)
}

//
func (c *request) WriteError(status int, err error) {
	res := &serverResponse{
		Version: Version,
		ID:      c.request.ID,
	}

	switch err := err.(type) {
	case *Error:
		res.Error = err
	default:
		res.Error = &Error{
			Code:    ErrServer,
			Message: err.Error(),
		}
	}

	c.writeServerResponse(res)
}

func (c *request) writeServerResponse(res *serverResponse) {
	c.writer.Header().Set(misc.HeaderXContentTypeOptions, "nosniff")
	// ID is null for notifications and they don't have a response.
	if c.request.ID != nil {
		c.writer.Header().Set(misc.HeaderContentType, misc.MIMEApplicationJSONCharsetUTF8)
		encoder := json.NewEncoder(c.encoder.Encode(c.writer))

		// Not sure in which case will this happen. But seems harmless.
		if err := encoder.Encode(res); err != nil {
			WriteError(c.writer, err)
		}
	}
}

// WriteError to ResponseWriter
func WriteError(w http.ResponseWriter, err error) {
	w.Header().Set(misc.HeaderXContentTypeOptions, "nosniff")
	w.Header().Set(misc.HeaderContentType, misc.MIMEApplicationJSONCharsetUTF8)
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `{"jsonrpc":%q,"id":1,"error":{"code":%d,"message":%q}}`,
		Version,
		ErrInvalidRequest,
		err.Error()) // ignore errors..
}
