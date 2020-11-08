package boltimore_test

import (
	"context"
	"testing"

	"github.com/draganm/bolted"
	"github.com/draganm/boltimore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestOnError(t *testing.T) {

	var err2 error
	var ec context.Context
	var m string
	var p string

	b, err := boltimore.Open(
		t.TempDir(),
		boltimore.ReadEndpoint("POST", "/ping/{xy}", func(ctx context.Context, tx bolted.WriteTx) error {
			return errors.New("failed")
		}),

		boltimore.ErrorListener(func(ctx context.Context, method, path string, err error) {
			err2 = err
			ec = ctx
			m = method
			p = path
		}),
	)
	require.NoError(t, err)

	defer b.Close()

	require.HTTPStatusCode(t, b.ServeHTTP, "POST", "/ping/z", nil, 500)
	require.EqualError(t, err2, "failed")
	require.Equal(t, "z", boltimore.RouteVariable(ec, "xy"))
	require.Equal(t, "POST", m)
	require.Equal(t, "/ping/{xy}", p)

}
