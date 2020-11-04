package boltimore_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/draganm/bolted"
	"github.com/draganm/boltimore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAddRead(t *testing.T) {

	t.Run("route variables", func(t *testing.T) {
		b, err := boltimore.Open(t.TempDir(), nil)
		require.NoError(t, err)

		defer b.Close()

		executed := false
		var xy string
		err = b.AddRead("POST", "/ping/{xy}", func(ctx context.Context, tx bolted.WriteTx) {
			executed = true
			xy = boltimore.RouteVariable(ctx, "xy")
		})

		require.NoError(t, err)

		require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping/z", nil, 201)
		require.True(t, executed)
		require.Equal(t, "z", xy)
	})

	t.Run("query values", func(t *testing.T) {
		b, err := boltimore.Open(t.TempDir(), nil)
		require.NoError(t, err)

		defer b.Close()

		executed := false

		var queryValues url.Values

		err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx) {
			executed = true
			queryValues = boltimore.QueryValues(ctx)
		})

		require.NoError(t, err)

		tw := newTestWriter()
		b.ServeHTTP(tw, &http.Request{
			Method: "POST",
			URL: &url.URL{
				Path:     "/ping",
				RawQuery: "foo=bar",
			},
			Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
		})

		require.Equal(t, 201, tw.status)
		require.True(t, executed)
		require.Equal(t, url.Values{"foo": []string{"bar"}}, queryValues)
	})

	t.Run("handling of request and response", func(t *testing.T) {

		t.Run("no input - no output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			err = b.AddRead("POST", "/ping", func(tx bolted.WriteTx) {
				executed = true
			})

			require.NoError(t, err)

			require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping", nil, 201)
			require.True(t, executed)
		})

		t.Run("context input - no output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false
			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx) {
				contextSet = ctx != nil
				executed = true
			})

			require.NoError(t, err)

			require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping", nil, 201)
			require.True(t, executed)
			require.True(t, contextSet)
		})

		t.Run("context and value input - no output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) {
				contextSet = ctx != nil
				executed = true
				input = i
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 201, tw.status)
		})

		t.Run("context and value input - error (nil returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) error {
				contextSet = ctx != nil
				executed = true
				input = i

				return nil
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 201, tw.status)
		})

		t.Run("context and value input - error (generic error returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) error {
				contextSet = ctx != nil
				executed = true
				input = i

				return errors.New("some err")
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 500, tw.status)
		})

		t.Run("context and value input - error (status code error returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) error {
				contextSet = ctx != nil
				executed = true
				input = i

				return boltimore.StatusCodeErr(404, "not found")
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 404, tw.status)
		})

		t.Run("context and value input - value and error (nil error returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			type outp struct {
				Bar string `json:"bar"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) (outp, error) {
				contextSet = ctx != nil
				executed = true
				input = i

				return outp{
					Bar: "baz",
				}, nil
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 200, tw.status)
			require.Equal(t, "{\"bar\":\"baz\"}\n", tw.String())
		})

		t.Run("context and value input - value and error (generic error returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			type outp struct {
				Bar string `json:"bar"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) (outp, error) {
				contextSet = ctx != nil
				executed = true
				input = i

				return outp{
					Bar: "baz",
				}, errors.New("failed")
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 500, tw.status)
		})

		t.Run("context and value input - value and error (status code error returned) output", func(t *testing.T) {
			b, err := boltimore.Open(t.TempDir(), nil)
			require.NoError(t, err)

			defer b.Close()

			executed := false
			contextSet := false

			type inp struct {
				Foo string `json:"foo"`
			}

			type outp struct {
				Bar string `json:"bar"`
			}

			var input inp

			err = b.AddRead("POST", "/ping", func(ctx context.Context, tx bolted.WriteTx, i inp) (outp, error) {
				contextSet = ctx != nil
				executed = true
				input = i

				return outp{
					Bar: "baz",
				}, boltimore.StatusCodeErr(201, "OK")
			})

			require.NoError(t, err)

			tw := newTestWriter()
			b.ServeHTTP(tw, &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/ping",
				},
				Body: ioutil.NopCloser(strings.NewReader(`{"foo": "bar"}`)),
			})

			require.True(t, executed)
			require.True(t, contextSet)
			require.Equal(t, inp{Foo: "bar"}, input)
			require.Equal(t, 201, tw.status)
			require.Equal(t, "OK\n", tw.String())
		})

	})

}
