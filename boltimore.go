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
	"go.uber.org/zap"
)

type Boltimore struct {
	*mux.Router
	DB      *bolted.Bolted
	cr      *cron.Cron
	Watcher *watcher.Watcher
	logger  *zap.SugaredLogger
}

type Option func(b *Boltimore) error

func Endpoint(method, path string, fn func(rc *RequestContext) error) Option {
	return Option(func(b *Boltimore) error {
		b.addEndpoint(method, path, fn)
		return nil
	})
}

type InitFunctionContext struct {
	DB     *bolted.Bolted
	Logger *zap.SugaredLogger
}

func InitFunction(fn func(ifc *InitFunctionContext) error) Option {
	return Option(func(b *Boltimore) error {
		return fn(&InitFunctionContext{
			DB:     b.DB,
			Logger: b.logger,
		})
	})
}

type CronFunctionContext struct {
	DB     *bolted.Bolted
	Logger *zap.SugaredLogger
}

func CronFunction(schedule string, fn func(cfc *CronFunctionContext)) Option {
	return Option(func(b *Boltimore) error {
		_, err := b.cr.AddFunc(schedule, func() {
			fn(&CronFunctionContext{
				DB:     b.DB,
				Logger: b.logger.With("cronFunction", schedule),
			})
		})

		if err != nil {
			return err
		}

		return nil
	})
}

type ChangeWatcherContext struct {
	DB     *bolted.Bolted
	Logger *zap.SugaredLogger
}

func ChangeWatcher(path string, fn func(cwc *ChangeWatcherContext)) Option {
	return Option(func(b *Boltimore) error {
		ch := make(chan struct{})
		go b.Watcher.WatchForChanges(context.Background(), path, func(c bolted.ReadTx) error {
			ch <- struct{}{}
			return nil
		})

		go func() {
			for range ch {
				fn(&ChangeWatcherContext{
					DB:     b.DB,
					Logger: b.logger.With("changeWatcher", path),
				})
			}
		}()
		return nil
	})
}

func ZapLogger(l *zap.SugaredLogger) Option {
	return Option(func(b *Boltimore) error {
		b.logger = l
		return nil
	})
}

func Open(dir string, options ...Option) (*Boltimore, error) {
	w := watcher.New()
	db, err := bolted.Open(filepath.Join(dir, "db"), 0700, bolted.WithChangeListeners(w))
	if err != nil {
		return nil, errors.Wrap(err, "while opening db")
	}
	return NewWithExistingDBAndWatcher(db, w, options...)
}

func NewWithExistingDBAndWatcher(db *bolted.Bolted, w *watcher.Watcher, options ...Option) (*Boltimore, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, errors.Wrap(err, "while creating initial ZAP logger")
	}

	b := &Boltimore{
		Router:  mux.NewRouter(),
		DB:      db,
		cr:      cron.New(),
		Watcher: w,
		logger:  logger.Sugar(),
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
	b.logger.Sync()
	b.cr.Stop()
	return b.DB.Close()
}
