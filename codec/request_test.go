package codec

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nspcc-dev/jsonrpc/misc"
	"github.com/stretchr/testify/require"
)

type (
	errorWriter  struct{}
	errorEncoder struct{}
)

func (errorWriter) Write(p []byte) (n int, err error)       { return 0, errors.New("error writer") }
func (errorEncoder) Encode(w http.ResponseWriter) io.Writer { return &errorWriter{} }

func TestCodecSuite(t *testing.T) {
	t.Run("Request codec test suite", func(t *testing.T) {
		t.Run("should create codec without errors", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "someMethod",
				"params": "params"
			}`))
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.NoError(t, err)

			require.Equal(t, "someMethod", r.Method())

			var args string
			require.NoError(t, r.ReadRequest(&args))
			require.Equal(t, "params", args)

			r.WriteResponse(args)

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":2.0,"id":1,"result":"params"}`, body)
		})

		t.Run("should fail with encoder error", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"jsonrpc": "2.0",
				"id": "1",
				"method": "someMethod",
				"params": "params"
			}`))
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.NoError(t, err)

			r.(*request).encoder = &errorEncoder{}

			require.Equal(t, "someMethod", r.Method())

			var args string
			require.NoError(t, r.ReadRequest(&args))
			require.Equal(t, "params", args)

			r.WriteResponse(args)

			body := strings.TrimSpace(rec.Body.String())
			require.Equal(t, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"error writer"}}`, body)
		})

		t.Run("should fail when method not POST", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.Nil(t, r)
			require.Error(t, err)

			our, ok := err.(*Error)
			require.True(t, ok)
			require.Equal(t, ErrInvalidRequest, our.Code)
			require.Equal(t, `rpc: POST method required, received GET`, our.Error())
		})

		t.Run("should fail with invalid json", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`invalid`))
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.Nil(t, r)
			require.Error(t, err)

			our, ok := err.(*Error)
			require.True(t, ok)
			require.Equal(t, ErrParse, our.Code)
		})

		t.Run("should fail on unknown version", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": ""}`))
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.Nil(t, r)
			require.Error(t, err)

			our, ok := err.(*Error)
			require.True(t, ok)
			require.Equal(t, ErrInvalidRequest, our.Code)
		})

		t.Run("should fail on bad params", func(t *testing.T) {
			var (
				rec   = httptest.NewRecorder()
				codec = NewCodec()
			)
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1, "params": 1}`))
			require.NoError(t, err)

			r, err := codec.NewRequest(rec, req)
			require.NoError(t, err)
			require.NotEmpty(t, r)

			var args string
			err = r.ReadRequest(&args)

			require.Error(t, err)
			require.Equal(t, "", args)
		})

		t.Run("HandleError suite", func(t *testing.T) {
			t.Run("should be false for nil error", func(t *testing.T) {
				var (
					rec   = httptest.NewRecorder()
					codec = NewCodec()
				)
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1}`))
				require.NoError(t, err)

				r, err := codec.NewRequest(rec, req)
				require.NoError(t, err)
				require.NotEmpty(t, r)

				require.False(t, r.HandleError(nil))
			})

			t.Run("should write misc.HTTPError", func(t *testing.T) {
				var (
					rec   = httptest.NewRecorder()
					codec = NewCodec()
				)
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1}`))
				require.NoError(t, err)

				r, err := codec.NewRequest(rec, req)
				require.NoError(t, err)
				require.NotEmpty(t, r)

				ok := r.HandleError(
					misc.NewHTTPError(http.StatusBadRequest, "bad request"))
				require.True(t, ok)
				body := strings.TrimSpace(rec.Body.String())
				require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"code=400, message=bad request"}}`, body)
			})

			t.Run("should write Error", func(t *testing.T) {
				var (
					rec   = httptest.NewRecorder()
					codec = NewCodec()
				)
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1}`))
				require.NoError(t, err)
				// Accept-Encoding: wrong, encoding
				req.Header.Set(misc.HeaderAcceptEncoding, "wrong, encoding")

				r, err := codec.NewRequest(rec, req)
				require.NoError(t, err)
				require.NotEmpty(t, r)

				ok := r.HandleError(&Error{
					Code:     ErrBadParams,
					Message:  "bad request",
					Internal: new(json.SyntaxError),
				})
				require.True(t, ok)
				body := strings.TrimSpace(rec.Body.String())
				require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32602,"message":"cannot unmarshal request"}}`, body)
			})

			t.Run("should write unknown type of error", func(t *testing.T) {
				var (
					rec   = httptest.NewRecorder()
					codec = NewCodec()
				)
				req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc": "2.0", "id": 1}`))
				require.NoError(t, err)

				// Accept-Encoding: gzip, deflate, br
				req.Header.Set(misc.HeaderAcceptEncoding, "gzip, deflate, br")

				r, err := codec.NewRequest(rec, req)
				require.NoError(t, err)
				require.NotEmpty(t, r)

				ok := r.HandleError(errors.New("bad request"))
				require.True(t, ok)
				body := strings.TrimSpace(rec.Body.String())
				require.Equal(t, `{"jsonrpc":2.0,"id":1,"error":{"code":-32000,"message":"bad request"}}`, body)
			})
		})
	})
}
