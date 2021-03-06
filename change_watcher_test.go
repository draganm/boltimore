package boltimore_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/draganm/bolted"
	"github.com/draganm/boltimore"
	"github.com/stretchr/testify/require"
)

func TestChangeWatcher(t *testing.T) {

	cnt := int64(0)

	b, err := boltimore.Open(t.TempDir(), boltimore.ChangeWatcher("/test", func(cwc *boltimore.ChangeWatcherContext) {
		atomic.AddInt64(&cnt, 1)
	}))

	require.NoError(t, err)

	for i := 0; i < 200; i++ {
		if atomic.LoadInt64(&cnt) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, int64(1), atomic.LoadInt64(&cnt))

	err = b.DB.Write(func(tx bolted.WriteTx) error {
		return tx.CreateMap("/test")
	})

	require.NoError(t, err)

	for i := 0; i < 200; i++ {
		if atomic.LoadInt64(&cnt) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, int64(2), atomic.LoadInt64(&cnt))

}
