package boltimore_test

import (
	"testing"
	"time"

	"github.com/draganm/bolted"
	"github.com/draganm/boltimore"
	"github.com/stretchr/testify/require"
)

func TestCron(t *testing.T) {
	waitChan := make(chan bool)

	b, err := boltimore.Open(t.TempDir(), boltimore.CronFunction("@every 1s", func(db *bolted.Bolted) error {
		close(waitChan)
		return nil
	}))
	require.NoError(t, err)
	defer b.Close()

	select {
	case <-waitChan:
		// this will happen when the cron function closes the channel
	case <-time.NewTimer(3 * time.Second).C:
		require.Fail(t, "timed out waiting for execution")
	}

}
