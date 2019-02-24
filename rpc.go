package jsonrpc

import (
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/go-helium/jsonrpc/codec"
	"github.com/go-helium/jsonrpc/misc"
	"github.com/pkg/errors"
)

type (
	// RPC server struct
	RPC struct {
		codec  *codecs
		method *methods
	}

	codecs struct {
		mu    *sync.RWMutex
		items map[string]codec.Interface
	}

	methods struct {
		mu    *sync.RWMutex
		items map[string]*method
	}

	method struct {
		method    reflect.Value // receiver method
		argsType  reflect.Type  // type of the request argument
		replyType reflect.Type  // type of the response argument
	}

	// CompressionSelector alias
	CompressionSelector = codec.CompressionSelector
)

var (
	// Precomputed the reflect.Type of error and http.Request
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfRequest = reflect.TypeOf((*http.Request)(nil)).Elem()

	errNotAFunction    = errors.New("method must be function")
	errNotEnoughArgs   = errors.New("method needs three args: *http.Request, *args, *reply")
	errNotEnoughOut    = errors.New("method needs one out: error")
	errNotReturnError  = errors.New("method needs one out: error")
	errFirstArgRequest = errors.New("method needs first parameter to be *http.Request")
	errSecondArgError  = errors.New("second argument must be a pointer and must be exported")
	errThirdArgError   = errors.New("third argument must be a pointer and must be exported")
)

// creates instance of codec registry
func newCodecRegistry() *codecs {
	return &codecs{
		mu:    new(sync.RWMutex),
		items: make(map[string]codec.Interface),
	}
}

// creates instance of methid registry
func newMethodRegistry() *methods {
	return &methods{
		mu:    new(sync.RWMutex),
		items: make(map[string]*method),
	}
}

// NewRPC create new server instance
func NewRPC() *RPC {
	return &RPC{
		codec:  newCodecRegistry(),
		method: newMethodRegistry(),
	}
}

// AddCodec register codec
func (s *RPC) AddCodec(codec codec.Interface, mime string) {
	s.codec.mu.Lock()
	defer s.codec.mu.Unlock()
	s.codec.items[strings.ToLower(mime)] = codec
}

// try to get codec or return error
func (s *RPC) getCodec(r *http.Request) (codec.Interface, error) {
	mime := r.Header.Get(misc.HeaderContentType)
	mime = strings.SplitAfterN(mime, ";", 1)[0]

	s.codec.mu.RLock()
	defer s.codec.mu.RUnlock()
	if result, ok := s.codec.items[strings.ToLower(mime)]; ok {
		return result, nil
	}
	return nil, misc.NewHTTPError(http.StatusUnsupportedMediaType, "rpc: unrecognized Content-Type: "+mime)
}

// AddMethod register method
// func(r *http.Request, args interface{}, reply *Reply) error
func (s *RPC) AddMethod(name string, fn interface{}) error {
	var (
		v     = reflect.ValueOf(fn)
		t     = reflect.TypeOf(fn)
		args  reflect.Type
		reply reflect.Type
	)

	if v.Kind() != reflect.Func {
		return errNotAFunction
	} else if t.NumIn() != 3 {
		return errNotEnoughArgs
	} else if t.NumOut() != 1 {
		return errNotEnoughOut
	}

	// Method must return error
	if rt := t.Out(0); rt != typeOfError {
		return errNotReturnError
	}

	// First argument must be *http.Request
	if rt := t.In(0); rt.Kind() != reflect.Ptr || rt.Elem() != typeOfRequest {
		return errFirstArgRequest
	}

	// Second argument must be exported or builtin.
	if args = t.In(1); !isExportedOrBuiltin(args) {
		return errSecondArgError
	}
	// Third argument must be a pointer and must be exported or builtin.
	if reply = t.In(2); !validateInputType(reply) {
		return errThirdArgError
	}

	s.method.mu.Lock()
	s.method.items[name] = &method{
		argsType:  args,
		replyType: reply.Elem(),
		method:    v,
	}
	s.method.mu.Unlock()

	return nil
}

// try to find and return method
func (s *RPC) get(name string) (*method, error) {
	s.method.mu.RLock()
	defer s.method.mu.RUnlock()
	if caller, ok := s.method.items[name]; ok {
		return caller, nil
	}
	return nil, &codec.Error{
		Code:    codec.ErrNoMethod,
		Message: "Method not found",
	}
}

// ServeHTTP implementation of http.Handler
func (s *RPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		cdc    codec.Interface
		req    codec.Request
		caller *method
	)

	if cdc, err = s.getCodec(r); err != nil {
		codec.WriteError(w, err)
		return
	} else if req, err = cdc.NewRequest(w, r); err != nil {
		codec.WriteError(w, err)
		return
	}

	defer func() { // catch internal errors:
		if err := recover(); err != nil {
			req.HandleError(&codec.Error{
				Code:    codec.ErrInternal,
				Message: "something went wrong",
				Data:    err,
			})
		}
	}()

	// Get method or return error
	if caller, err = s.get(req.Method()); req.HandleError(err) {
		return
	}

	// Decode the args.
	args := reflect.New(caller.argsType)
	if err := req.ReadRequest(args.Interface()); req.HandleError(err) {
		return
	}

	// Call the service method.
	reply := reflect.New(caller.replyType)
	errValue := caller.method.Call([]reflect.Value{
		reflect.ValueOf(r),
		args.Elem(),
		reply,
	})

	// Cast the result to error if needed.
	if errValue[0].Interface() != nil && req.HandleError(errValue[0].Interface().(error)) {
		return
	}

	req.WriteResponse(reply.Interface())
}

// isExported returns true of a string is an exported (upper case) name.
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func validateInputType(t reflect.Type) bool {
	return (t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice) &&
		isExportedOrBuiltin(t)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin.
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}
