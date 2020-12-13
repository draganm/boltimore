package boltimore

import (
	"encoding/json"
	"net/http"

	"github.com/draganm/bolted"
	"github.com/draganm/bolted/watcher"
	"github.com/gorilla/mux"
)

type RequestContext struct {
	Request         *http.Request
	ResponseWriter  http.ResponseWriter
	DB              *bolted.Bolted
	responseWritten bool
	Watcher         *watcher.Watcher
}

func (r *RequestContext) RouteVariable(name string) string {
	req := r.Request
	vars := mux.Vars(req)
	return vars[name]
}

func (r *RequestContext) ParseJSON(v interface{}) error {
	return json.NewDecoder(r.Request.Body).Decode(v)
}

func (r *RequestContext) RespondWithJSON(v interface{}) error {
	r.ResponseWriter.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(r.ResponseWriter).Encode(v)
}

func (b *Boltimore) addEndpoint(method, path string, action func(rc *RequestContext) error) {

	b.Router.Methods(method).Path(path).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rc := &RequestContext{
			Request:        req,
			ResponseWriter: w,
			DB:             b.DB,
			Watcher:        b.Watcher,
		}

		err := action(rc)

		if err != nil {
			if !rc.responseWritten {
				http.Error(w, "internal server error", 500)
			}
		}
	})
}
