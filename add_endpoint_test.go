package boltimore_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/draganm/boltimore"
	"github.com/stretchr/testify/require"
)

type testWriter struct {
	h http.Header
	*bytes.Buffer
	status int
}

func (tw *testWriter) Header() http.Header {
	return tw.h
}

func (tw *testWriter) WriteHeader(statusCode int) {
	tw.status = statusCode
}

func newTestWriter() *testWriter {
	return &testWriter{
		h:      http.Header{},
		Buffer: new(bytes.Buffer),
		status: 200,
	}
}

func TestAddEndpoint(t *testing.T) {

	t.Run("route variables", func(t *testing.T) {
		executed := false
		var xy string

		b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping/{xy}", func(rc *boltimore.RequestContext) error {
			executed = true
			xy = rc.RouteVariable("xy")
			return nil
		}))
		require.NoError(t, err)

		defer b.Close()

		require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping/z", nil, 200)
		require.True(t, executed)
		require.Equal(t, "z", xy)
	})

	t.Run("handling of request and response", func(t *testing.T) {

		t.Run("no input - no output", func(t *testing.T) {
			executed := false

			b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
				executed = true
				return nil
			}))
			require.NoError(t, err)

			defer b.Close()

			require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping", nil, 200)
			require.True(t, executed)
		})

		t.Run("parsing json request", func(t *testing.T) {

			type inp struct {
				Foo string `json:"foo"`
			}

			in := inp{}

			b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
				return rc.ParseJSON(&in)
			}))

			require.NoError(t, err)

			defer b.Close()

			tw := newTestWriter()

			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.Equal(t, inp{Foo: "bar"}, in)
			require.Equal(t, 200, tw.status)

		})

		t.Run("writing JSON response", func(t *testing.T) {

			type inp struct {
				Foo string `json:"foo"`
			}

			in := inp{}

			b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
				return rc.ParseJSON(&in)
			}))

			require.NoError(t, err)

			defer b.Close()

			tw := newTestWriter()

			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.Equal(t, inp{Foo: "bar"}, in)
			require.Equal(t, 200, tw.status)

		})

		// t.Run("context and value input - no output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	var input inp

		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i
		// 	}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	require.NoError(t, err)

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 201, tw.status)
		// })

		// t.Run("context and value input - error (nil returned) output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	var input inp

		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i

		// 		return nil
		// 	}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	require.NoError(t, err)

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 201, tw.status)
		// })

		// t.Run("context and value input - error (generic error returned) output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	var input inp
		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i

		// 		return errors.New("some err")
		// 	}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	require.NoError(t, err)

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 500, tw.status)
		// })

		// t.Run("context and value input - error (status code error returned) output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	var input inp

		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i

		// 		return boltimore.StatusCodeErr(404, "not found")
		// 	}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	require.NoError(t, err)

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 404, tw.status)
		// })

		// t.Run("context and value input - value and error (nil error returned) output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	type outp struct {
		// 		Bar string `json:"bar"`
		// 	}

		// 	var input inp

		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint(
		// 		"POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 			contextSet = ctx != nil
		// 			executed = true
		// 			input = i

		// 			return outp{
		// 				Bar: "baz",
		// 			}, nil
		// 		}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 200, tw.status)
		// 	require.Equal(t, "{\"bar\":\"baz\"}\n", tw.String())
		// })

		// t.Run("context and value input - value and error (generic error returned) output", func(t *testing.T) {
		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	type outp struct {
		// 		Bar string `json:"bar"`
		// 	}

		// 	var input inp
		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i

		// 		return outp{
		// 			Bar: "baz",
		// 		}, errors.New("failed")
		// 	}))

		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 500, tw.status)
		// })

		// t.Run("context and value input - value and error (status code error returned) output", func(t *testing.T) {

		// 	executed := false
		// 	contextSet := false

		// 	type inp struct {
		// 		Foo string `json:"foo"`
		// 	}

		// 	type outp struct {
		// 		Bar string `json:"bar"`
		// 	}

		// 	var input inp

		// 	b, err := boltimore.Open(t.TempDir(), boltimore.Endpoint("POST", "/ping", func(rc *boltimore.RequestContext) error {
		// 		contextSet = ctx != nil
		// 		executed = true
		// 		input = i

		// 		return outp{
		// 			Bar: "baz",
		// 		}, boltimore.StatusCodeErr(201, "OK")
		// 	}))
		// 	require.NoError(t, err)

		// 	defer b.Close()

		// 	require.NoError(t, err)

		// 	tw := newTestWriter()
		// 	b.ServeHTTP(tw, &http.Request{
		// 		Method: "POST",
		// 		URL: &url.URL{
		// 			Path: "/ping",
		// 		},
		// 		Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		// 	})

		// 	require.True(t, executed)
		// 	require.True(t, contextSet)
		// 	require.Equal(t, inp{Foo: "bar"}, input)
		// 	require.Equal(t, 201, tw.status)
		// 	require.Equal(t, "OK\n", tw.String())
		// })

	})

}
