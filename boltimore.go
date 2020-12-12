package boltimore

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/draganm/bolted"
	"github.com/draganm/bolted/watcher"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

type Boltimore struct {
	*mux.Router
	DB      *bolted.Bolted
	cr      *cron.Cron
	Watcher *watcher.Watcher
}

type Option func(b *Boltimore) error

func Endpoint(method, path string, fn func(rc *RequestContext) error) Option {
	return Option(func(b *Boltimore) error {
		b.addEndpoint(method, path, fn)
		return nil
	})
}

func InitFunction(fn func(tx bolted.WriteTx) error) Option {
	return Option(func(b *Boltimore) error {
		return b.DB.Write(func(tx bolted.WriteTx) error {
			return fn(tx)
		})
	})
}

func DBInitFunction(fn func(db *bolted.Bolted) error) Option {
	return Option(func(b *Boltimore) error {
		return fn(b.DB)
	})
}

func CronFunction(schedule string, fn func(db *bolted.Bolted)) Option {
	return Option(func(b *Boltimore) error {
		_, err := b.cr.AddFunc(schedule, func() {
			fn(b.DB)
		})

		if err != nil {
			return err
		}

		return nil
	})
}

func ChangeWatcher(path string, fn func(db *bolted.Bolted)) Option {
	return Option(func(b *Boltimore) error {
		ch := make(chan struct{})
		go b.Watcher.WatchForChanges(context.Background(), path, func(c bolted.ReadTx) error {
			ch <- struct{}{}
			return nil
		})

		go func() {
			for range ch {
				fn(b.DB)
			}
		}()
		return nil
	})
}

func Open(dir string, options ...Option) (*Boltimore, error) {
	w := watcher.New()
	db, err := bolted.Open(filepath.Join(dir, "db"), 0700, bolted.WithChangeListeners(w))
	if err != nil {
		return nil, errors.Wrap(err, "while opening db")
	}

	b := &Boltimore{
		Router:  mux.NewRouter(),
		DB:      db,
		cr:      cron.New(),
		Watcher: w,
	}

	for _, o := range options {
		err = o(b)
		if err != nil {
			db.Close()
			return nil, err
		}
	}

	b.cr.Start()

	return b, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var boltedWriteTxType = reflect.TypeOf((*bolted.WriteTx)(nil)).Elem()
var boltedReadTxType = reflect.TypeOf((*bolted.ReadTx)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *Boltimore) Close() error {
	b.cr.Stop()
	return b.DB.Close()
}
