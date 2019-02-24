package jsonrpc

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-helium/jsonrpc/codec"
	"github.com/go-helium/jsonrpc/misc"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
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
	Convey("RPC Test Suite", t, func() {
		var (
			rec = httptest.NewRecorder()
			srv = NewRPC()
		)

		Convey("should run without errors", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})

			So(err, ShouldBeNil)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"result":10}`)
		})

		Convey("should recover panic in method", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				panic("panic error")
			})

			So(err, ShouldBeNil)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32603,"message":"something went wrong","data":"panic error"}}`)
		})

		Convey("should fail with GET request", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			req, err := http.NewRequest(http.MethodGet, "", nil)
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"rpc: POST method required, received GET"}}`)
		})

		Convey("should fail with wrong args", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})

			So(err, ShouldBeNil)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": "a"
			}`))
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32602,"message":"cannot unmarshal request","data":"a"}}`)
		})

		Convey("should return error from handler", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				return errors.New("method error")
			})

			So(err, ShouldBeNil)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1]
			}`))
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"method error"}}`)
		})

		Convey("should fail with Method not found", func() {
			cdc := codec.NewCustom(&CompressionSelector{})
			srv.AddCodec(cdc, misc.MIMEApplicationJSON)
			srv.AddCodec(cdc, misc.MIMEApplicationJSONCharsetUTF8)

			req, err := http.NewRequest(http.MethodPost, "", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "unknown_method",
				"params": [1,2,3,4]
			}`))
			So(err, ShouldBeNil)

			req.Header.Set(misc.HeaderContentType,
				misc.MIMEApplicationJSONCharsetUTF8)

			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32601,"message":"Method not found"}}`)
		})

		Convey("should fail on unknown content-type", func() {
			err := srv.AddMethod("sum", func(r *http.Request, args []int, reply *int) error {
				for i := range args {
					*reply += args[i]
				}
				return nil
			})
			So(err, ShouldBeNil)

			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "sum",
				"params": [1,2,3,4]
			}`))
			So(err, ShouldBeNil)
			So(func() { srv.ServeHTTP(rec, req) }, ShouldNotPanic)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"code=415, message=rpc: unrecognized Content-Type: "}}`)
		})

		Convey("should fail on bad method", func() {
			Convey("method must be function", func() {
				err := srv.AddMethod("sum", new(int))
				So(err, ShouldBeError, errNotAFunction)
			})

			Convey("not enough args", func() {
				err := srv.AddMethod("sum", func() {})
				So(err, ShouldBeError, errNotEnoughArgs)
			})

			Convey("must return error", func() {
				err := srv.AddMethod("sum", func(a, b, c int) {})
				So(err, ShouldBeError, errNotEnoughOut)
				err = srv.AddMethod("sum", func(a, b, c int) int { return 0 })
				So(err, ShouldBeError, errNotReturnError)
			})

			Convey("first arg error", func() {
				err := srv.AddMethod("sum", func(a, b, c int) error { return nil })
				So(err, ShouldBeError, errFirstArgRequest)
			})

			Convey("second arg error", func() {
				err := srv.AddMethod("sum", func(*http.Request, fakeType, int) error { return nil })
				So(err, ShouldBeError, errSecondArgError)
			})

			Convey("third arg error", func() {
				err := srv.AddMethod("sum", func(*http.Request, **struct{}, int) error { return nil })
				So(err, ShouldBeError, errThirdArgError)
			})
		})
	})
}
