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
	db *bolted.Bolted
}

func Open(dir string, init func(tx bolted.WriteTx) error) (*Boltimore, error) {
	db, err := bolted.Open(filepath.Join(dir, "db"), 0700)
	if err != nil {
		return nil, errors.Wrap(err, "while opening db")
	}

	if init != nil {
		err = db.Write(init)
		if err != nil {
			return nil, errors.Wrap(err, "while executing init")
		}
	}

	return &Boltimore{
		Router: mux.NewRouter(),
		db:     db,
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var boltedWriteTxType = reflect.TypeOf((*bolted.WriteTx)(nil)).Elem()
var boltedReadTxType = reflect.TypeOf((*bolted.ReadTx)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *Boltimore) Close() error {
	return b.db.Close()
}
