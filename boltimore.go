package boltimore

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/draganm/bolted"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type Boltimore struct {
	*mux.Router
	db            *bolted.Bolted
	initFunctions [](func(tx bolted.WriteTx) error)
}

type Option func(b *Boltimore) error

func ReadEndpoint(method, path string, fn interface{}) Option {
	return Option(func(b *Boltimore) error {
		return b.addRead(method, path, fn)
	})
}

func WriteEndpoint(method, path string, fn interface{}) Option {
	return Option(func(b *Boltimore) error {
		return b.addWrite(method, path, fn)
	})
}
func InitFunction(fn func(tx bolted.WriteTx) error) Option {
	return Option(func(b *Boltimore) error {
		b.initFunctions = append(b.initFunctions, fn)
		return nil
	})
}

func Open(dir string, options ...Option) (*Boltimore, error) {
	db, err := bolted.Open(filepath.Join(dir, "db"), 0700)
	if err != nil {
		return nil, errors.Wrap(err, "while opening db")
	}

	b := &Boltimore{
		Router: mux.NewRouter(),
		db:     db,
	}

	for _, o := range options {
		err = o(b)
		if err != nil {
			db.Close()
			return nil, err
		}
	}

	for _, init := range b.initFunctions {
		err = db.Write(init)
		if err != nil {
			return nil, errors.Wrap(err, "while executing init")
		}
	}

	return b, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var boltedWriteTxType = reflect.TypeOf((*bolted.WriteTx)(nil)).Elem()
var boltedReadTxType = reflect.TypeOf((*bolted.ReadTx)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *Boltimore) Close() error {
	return b.db.Close()
}
