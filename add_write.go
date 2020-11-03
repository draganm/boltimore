package boltimore

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/draganm/bolted"
	"github.com/pkg/errors"
)

func (b *Boltimore) AddWrite(method, path string, fn interface{}) error {
	fnv := reflect.ValueOf(fn)
	fnt := fnv.Type()

	if fnt.Kind() != reflect.Func {
		return errors.New("not a function")
	}

	var inParams func(r *http.Request, wtx bolted.WriteTx) ([]reflect.Value, error)

	var createContext = func(r *http.Request) reflect.Value {
		ctx := context.WithValue(r.Context(), boltimoreCtx, &BoltimoreContext{
			req: r,
		})
		return reflect.ValueOf(ctx)

	}

	var hasInput = false

	switch fnt.NumIn() {
	case 0:
		return errors.New("function must accept at least one argument")
	case 1:
		if !fnt.In(0).AssignableTo(boltedWriteTxType) {
			return errors.New("function must accept bolted.WriteTx as the first argument")
		}
		inParams = func(r *http.Request, wtx bolted.WriteTx) ([]reflect.Value, error) {
			return []reflect.Value{reflect.ValueOf(wtx)}, nil
		}
	case 2:
		if !fnt.In(0).AssignableTo(contextType) {
			return errors.New("function must accept context.Context as the first argument")
		}
		if !fnt.In(1).AssignableTo(boltedWriteTxType) {
			return errors.New("function must accept bolted.WriteTx as the second argument")
		}
		inParams = func(r *http.Request, wtx bolted.WriteTx) ([]reflect.Value, error) {
			return []reflect.Value{createContext(r), reflect.ValueOf(wtx)}, nil
		}
	case 3:
		if !fnt.In(0).AssignableTo(contextType) {
			return errors.New("function must accept context.Context as the first argument")
		}
		if !fnt.In(1).AssignableTo(boltedWriteTxType) {
			return errors.New("function must accept bolted.WriteTx as the second argument")
		}
		inParams = func(r *http.Request, wtx bolted.WriteTx) ([]reflect.Value, error) {
			reqT := fnt.In(2)
			reqInstance := reflect.New(reqT)
			err := json.NewDecoder(r.Body).Decode(reqInstance.Interface())

			if err != nil {
				return nil, err
			}

			return []reflect.Value{createContext(r), reflect.ValueOf(wtx), reqInstance.Elem()}, nil
		}

	default:
		return errors.New("function can have at most 2 parameters")
	}

	var writeResponse func(res []reflect.Value, w http.ResponseWriter) error

	switch fnt.NumOut() {
	case 0:
		writeResponse = func(res []reflect.Value, w http.ResponseWriter) error {
			w.WriteHeader(201)
			return nil
		}
	case 1:
		if fnt.Out(0) != errorType {
			return errors.New("function must return error as only return value")
		}

		writeResponse = func(res []reflect.Value, w http.ResponseWriter) error {

			v := res[0]

			if v.Interface() != nil {
				err := v.Interface().(error)

				sce, isStatusCode := err.(statusCodeError)
				if isStatusCode {
					http.Error(w, sce.message, sce.statusCode)
					return err
				}

				http.Error(w, "internal server error", 500)
				return err
			}
			w.WriteHeader(201)
			return nil
		}
	case 2:
		if fnt.Out(1) != errorType {
			return errors.New("function must return error the last return value")
		}

		writeResponse = func(res []reflect.Value, w http.ResponseWriter) error {

			v := res[0]
			errv := res[1]

			if errv.Interface() != nil {
				err := errv.Interface().(error)

				sce, isStatusCode := err.(statusCodeError)
				if isStatusCode {
					http.Error(w, sce.message, sce.statusCode)
					return err
				}

				http.Error(w, "internal server error", 500)
				return err
			}

			w.Header().Set("Content-Type", "application/json")

			err := json.NewEncoder(w).Encode(v.Interface())
			if err != nil {
				return err
			}

			// w.WriteHeader(201)
			return nil
		}

	default:
		return errors.New("function can have at most 2 parameters")
	}

	b.Router.Methods(method).Path(path).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasInput && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "request must be JSON", 400)
			return
		}

		b.db.Write(func(tx bolted.WriteTx) error {
			inParams, err := inParams(r, tx)
			if err != nil {
				http.Error(w, "internal server error", 500)
				return err
			}
			res := fnv.Call(inParams)
			return writeResponse(res, w)
		})

	})

	return nil

}
