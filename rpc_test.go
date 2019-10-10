package jsonrpc

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nspcc-dev/jsonrpc/codec"
	"github.com/nspcc-dev/jsonrpc/misc"
	"github.com/stretchr/testify/require"
)

type fakeType struct{}

func (fakeType) Align() int {
	panic("implement me")
}

func (fakeType) FieldAlign() int {
	panic("implement me")
}

func (fakeType) Method(int) reflect.Method {
	panic("implement me")
}

func (fakeType) MethodByName(string) (reflect.Method, bool) {
	panic("implement me")
}

func (fakeType) NumMethod() int {
	panic("implement me")
}

func (fakeType) Name() string {
	panic("implement me")
}

func (fakeType) PkgPath() string {
	return "________"
}

func (fakeType) Size() uintptr {
	panic("implement me")
}

func (fakeType) String() string {
	panic("implement me")
}

func (fakeType) Kind() reflect.Kind {
	panic("implement me")
}

func (fakeType) Implements(u reflect.Type) bool {
	panic("implement me")
}

func (fakeType) AssignableTo(u reflect.Type) bool {
	panic("implement me")
}

func (fakeType) ConvertibleTo(u reflect.Type) bool {
	panic("implement me")
}

func (fakeType) Comparable() bool {
	panic("implement me")
}

func (fakeType) Bits() int {
	panic("implement me")
}

func (fakeType) ChanDir() reflect.ChanDir {
	panic("implement me")
}

func (fakeType) IsVariadic() bool {
	panic("implement me")
}

func (fakeType) Elem() reflect.Type {
	panic("implement me")
}

func (fakeType) Field(i int) reflect.StructField {
	panic("implement me")
}

func (fakeType) FieldByIndex(index []int) reflect.StructField {
	panic("implement me")
}

func (fakeType) FieldByName(name string) (reflect.StructField, bool) {
	panic("implement me")
}

func (fakeType) FieldByNameFunc(match func(string) bool) (reflect.StructField, bool) {
	panic("implement me")
}

func (fakeType) In(i int) reflect.Type {
	panic("implement me")
}

func (fakeType) Key() reflect.Type {
	panic("implement me")
}

func (fakeType) Len() int {
	panic("implement me")
}

func (fakeType) NumField() int {
	panic("implement me")
}

func (fakeType) NumIn() int {
	panic("implement me")
}

func (fakeType) NumOut() int {
	panic("implement me")
}

func (fakeType) Out(i int) reflect.Type {
	panic("implement me")
}

func (fakeType) common() *interface{} {
	panic("implement me")
}

func (fakeType) uncommon() *interface{} {
	panic("implement me")
}

func TestRPCSuite(t *testing.T) {
	t.Run("RPC Test Suite", func(t *testing.T) {
		t.Run("should run without errors", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})

			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			// Accept-Encoding: gzip, deflate, br
			req.Header.Set(misc.HeaderAcceptEncoding, "gzip, deflate, br")

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			gz, err := gzip.NewReader(rec.Body)
			require.NoError(t, err)

			body, err := ioutil.ReadAll(gz)
			require.NoError(t, err)

			body = bytes.TrimSpace(body)
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"result":10}`, string(body))
		})

		t.Run("should recover panic in method", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				panic("panic error")
			})

			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			req.Header.Set(misc.HeaderAcceptEncoding, "wrong, encoding")

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32603,"message":"something went wrong","data":"panic error"}}`, body)
		})

		t.Run("should fail with GET request", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			req, err := http.NewRequest(http.MethodGet, "", nil)
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			req.Header.Set(misc.HeaderAcceptEncoding, "deflate, gzip")

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			fl := flate.NewReader(rec.Body)

			body, err := ioutil.ReadAll(fl)
			require.NoError(t, err)

			body = bytes.TrimSpace(body)

			require.Equal(t, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"rpc: POST method required, received GET"}}`, string(body))
		})

		t.Run("should fail with wrong args", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})

			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": "a"
			}`))
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32602,"message":"cannot unmarshal request","data":"a"}}`, body)
		})

		t.Run("should return error from handler", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				return Error("method error")
			})

			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1]
			}`))
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"method error"}}`, body)
		})

		t.Run("should fail with Method not found", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "unknown_method",
				"params": [1,2,3,4]
			}`))
			require.NoError(t, err)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32601,"message":"Method not found"}}`, body)
		})

		t.Run("should fail on unknown content-type", func(t *testing.T) {
			var (
				rec = httptest.NewRecorder()
				srv = NewRPC()
			)
			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			require.NoError(t, err)

			require.NotPanics(t, func() { srv.ServeHTTP(rec, req) })

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"code=415, message=rpc: unrecognized Content-Type: "}}`, body)
		})

		t.Run("should fail on bad method", func(t *testing.T) {
			t.Run("method must be function", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", new(int))
				require.EqualError(t, err, ErrNotAFunction.Error())
			})

			t.Run("not enough args", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", func() {})
				require.EqualError(t, err, ErrNotEnoughArgs.Error())
			})

			t.Run("must return error", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", func(a, b, c int) {})
				require.EqualError(t, err, ErrNotEnoughOut.Error())
				err = srv.AddMethod("sum", func(a, b, c int) int { return 0 })
				require.EqualError(t, err, ErrNotReturnError.Error())
			})

			t.Run("first arg error", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", func(a, b, c int) error { return nil })
				require.EqualError(t, err, ErrFirstArgRequest.Error())
			})

			t.Run("second arg error", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", func(*http.Request, fakeType, int) error { return nil })
				require.EqualError(t, err, ErrSecondArgError.Error())
			})

			t.Run("third arg error", func(t *testing.T) {
				var srv = NewRPC()
				err := srv.AddMethod("sum", func(*http.Request, **struct{}, int) error { return nil })
				require.EqualError(t, err, ErrThirdArgError.Error())
			})
		})
	})
}
