package boltimore

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

type key string

const boltimoreCtx key = "boltimore_context"

type BoltimoreContext struct {
	req *http.Request
}

func RouteVariable(ctx context.Context, key string) string {
	bctx := ctx.Value(boltimoreCtx)
	bc, valueSet := bctx.(*BoltimoreContext)

	if !valueSet {
		return ""
	}

	return mux.Vars(bc.req)[key]
}

func QueryValues(ctx context.Context) url.Values {
	bctx := ctx.Value(boltimoreCtx)
	bc, valueSet := bctx.(*BoltimoreContext)

	if !valueSet {
		return url.Values{}
	}

	return bc.req.URL.Query()
}
