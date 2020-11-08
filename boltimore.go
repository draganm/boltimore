package boltimore

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/draganm/bolted"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

type Boltimore struct {
	*mux.Router
	db             *bolted.Bolted
	initFunctions  [](func(tx bolted.WriteTx) error)
	cr             *cron.Cron
	errorListeners []func(context.Context, string, string, error)
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

func CronFunction(schedule string, fn func(db *bolted.Bolted)) Option {
	return Option(func(b *Boltimore) error {
		_, err := b.cr.AddFunc(schedule, func() {
			fn(b.db)
		})

		if err != nil {
			return err
		}

		return nil
	})
}

func ErrorListener(fn func(context.Context, string, string, error)) Option {
	return Option(func(b *Boltimore) error {
		b.errorListeners = append(b.errorListeners, fn)
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
		cr:     cron.New(),
	}

	for _, o := range options {
		err = o(b)
		if err != nil {
			db.Close()
			return nil, err
		}
	}

	b.cr.Start()

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
	b.cr.Stop()
	return b.db.Close()
}

func (b *Boltimore) onError(ctx context.Context, method, path string, err error) {
	for _, el := range b.errorListeners {
		el(ctx, method, path, err)
	}
}
