package codec

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-helium/jsonrpc/misc"
	. "github.com/smartystreets/goconvey/convey"
)

type (
	errorWriter  struct{}
	errorEncoder struct{}
)

func (errorWriter) Write(p []byte) (n int, err error)       { return 0, errors.New("error writer") }
func (errorEncoder) Encode(w http.ResponseWriter) io.Writer { return &errorWriter{} }

func TestCodecSuite(t *testing.T) {
	Convey("Request codec test suite", t, func() {
		var (
			rec   = httptest.NewRecorder()
			codec = NewCodec()
		)

		Convey("should create codec without errors", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "someMethod",
				"params": "params"
			}`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(err, ShouldBeNil)

			So(r.Method(), ShouldEqual, "someMethod")

			var args string
			So(r.ReadRequest(&args), ShouldBeNil)
			So(args, ShouldEqual, "params")

			r.WriteResponse(args)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"result":"params"}`)
		})

		Convey("should fail with encoder error", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "someMethod",
				"params": "params"
			}`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(err, ShouldBeNil)

			r.(*request).encoder = &errorEncoder{}

			So(r.Method(), ShouldEqual, "someMethod")

			var args string
			So(r.ReadRequest(&args), ShouldBeNil)
			So(args, ShouldEqual, "params")

			r.WriteResponse(args)

			body := strings.TrimSpace(rec.Body.String())
			So(body, ShouldEqual, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"error writer"}}`)
		})

		Convey("should fail when method not POST", func() {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(r, ShouldBeNil)
			So(err, ShouldBeError)

			our, ok := err.(*Error)
			So(ok, ShouldBeTrue)
			So(our.Code, ShouldEqual, ErrInvalidRequest)
			So(our.Error(), ShouldEqual, `rpc: POST method required, received GET`)
		})

		Convey("should fail with invalid json", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`invalid`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(r, ShouldBeNil)
			So(err, ShouldBeError)

			our, ok := err.(*Error)
			So(ok, ShouldBeTrue)
			So(our.Code, ShouldEqual, ErrParse)
		})

		Convey("should fail on unknown version", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": ""}`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(r, ShouldBeNil)
			So(err, ShouldBeError)

			our, ok := err.(*Error)
			So(ok, ShouldBeTrue)
			So(our.Code, ShouldEqual, ErrInvalidRequest)
		})

		Convey("should fail on bad params", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1, "params": 1}`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(err, ShouldBeNil)
			So(r, ShouldNotBeNil)

			var args string
			err = r.ReadRequest(&args)

			So(err, ShouldBeError)
			So(args, ShouldEqual, "")
		})

		Convey("HandleError suite", func() {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1}`))
			So(err, ShouldBeNil)

			r, err := codec.NewRequest(rec, req)
			So(r, ShouldNotBeNil)
			So(err, ShouldBeNil)

			Convey("should be false for nil error", func() {
				ok := r.HandleError(nil)
				So(ok, ShouldBeFalse)
			})

			Convey("should write misc.HTTPError", func() {
				ok := r.HandleError(
					misc.NewHTTPError(http.StatusBadRequest, "bad request"))
				So(ok, ShouldBeTrue)
				body := strings.TrimSpace(rec.Body.String())
				So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"code=400, message=bad request"}}`)
			})

			Convey("should write Error", func() {
				ok := r.HandleError(&Error{
					Code:     ErrBadParams,
					Message:  "bad request",
					Internal: new(json.SyntaxError),
				})
				So(ok, ShouldBeTrue)
				body := strings.TrimSpace(rec.Body.String())
				So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32602,"message":"cannot unmarshal request"}}`)
			})

			Convey("should write unknown type of error", func() {
				ok := r.HandleError(errors.New("bad request"))
				So(ok, ShouldBeTrue)
				body := strings.TrimSpace(rec.Body.String())
				So(body, ShouldEqual, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"bad request"}}`)
			})
		})
	})
}
