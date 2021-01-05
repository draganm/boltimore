package backup_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/draganm/bolted"
	"github.com/draganm/boltimore"
	"github.com/draganm/boltimore/backup"
	"github.com/stretchr/testify/require"
)

func TestBackupAndRestore(t *testing.T) {
	b, err := boltimore.Open(
		t.TempDir(),
		boltimore.Endpoint("GET", "/backup", backup.BackupEndpoint),
		boltimore.Endpoint("PUT", "/backup", backup.RestoreEndpoint),
	)
	require.NoError(t, err)

	defer b.Close()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	s := &http.Server{
		Handler: b.Router,
	}

	go s.Serve(l)
	defer s.Close()

	err = b.DB.Write(func(tx bolted.WriteTx) error {
		err = tx.CreateMap("foo")
		if err != nil {
			return err
		}

		err = tx.Put("foo/bar", []byte{1, 2, 3})
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	backupURL := fmt.Sprintf("http://%s/backup", l.Addr().String())

	res, err := http.Get(backupURL)
	require.NoError(t, err)

	defer res.Body.Close()
	require.Equal(t, 200, res.StatusCode)

	backup, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	err = b.DB.Write(func(tx bolted.WriteTx) error {
		err = tx.Delete("foo")
		if err != nil {
			return err
		}

		return tx.CreateMap("baz")
	})

	require.NoError(t, err)

	req, err := http.NewRequest("PUT", backupURL, bytes.NewReader(backup))
	require.NoError(t, err)

	res2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer res2.Body.Close()
	require.Equal(t, 200, res2.StatusCode)

	maps := []string{}

	err = b.DB.Read(func(tx bolted.ReadTx) error {
		it, err := tx.Iterator("")
		if err != nil {
			return err
		}

		for ; !it.Done; it.Next() {
			maps = append(maps, it.Key)
		}

		return nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"foo"}, maps)
}
