package backup

import (
	"io"
	"path"

	"github.com/draganm/bolted"
	"github.com/draganm/bolted/dump"
	"github.com/draganm/boltimore"
	"github.com/pkg/errors"
)

func BackupEndpoint(rc *boltimore.RequestContext) (err error) {

	defer func() {
		if err != nil {
			rc.Logger.With("error", err).Info("while writing backup")
		}
	}()

	return rc.DB.Read(func(tx bolted.ReadTx) error {
		tw := dump.NewWriter(rc.ResponseWriter)
		toDo := []string{""}
		for len(toDo) > 0 {
			head := toDo[0]
			toDo = toDo[1:]

			if head != "" {
				_, err = tw.CreateMap(head)
				if err != nil {
					return err
				}
			}

			if err != nil {
				errors.Wrapf(err, "while writing dir header for %s", head)
			}

			it, err := tx.Iterator(head)
			if err != nil {
				return errors.Wrapf(err, "while creating iterator for %s", head)
			}

			for ; !it.Done; it.Next() {
				pth := path.Join(head, it.Key)

				if it.Value == nil {
					toDo = append(toDo, pth)
					continue
				}

				_, err = tw.Put(path.Join(head, it.Key), it.Value)
				if err != nil {
					return err
				}

			}
		}
		return nil
	})
}

func RestoreEndpoint(rc *boltimore.RequestContext) (err error) {

	defer func() {
		if err != nil {
			rc.Logger.With("error", err).Info("while restoring backup")
		}
	}()

	return rc.DB.Write(func(tx bolted.WriteTx) error {
		it, err := tx.Iterator("")
		if err != nil {
			return err
		}

		keysToDelete := []string{}

		for ; !it.Done; it.Next() {
			keysToDelete = append(keysToDelete, it.Key)
		}

		for _, k := range keysToDelete {
			err = tx.Delete(k)
			if err != nil {
				return errors.Wrapf(err, "while deleting %s", k)
			}
		}

		tr := dump.NewReader(rc.Request.Body)

		for {
			nx, err := tr.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				return errors.Wrap(err, "while reading next tar entry")
			}

			switch nx.Type {
			case dump.Put:
				err = tx.Put(nx.Key, nx.Value)
				if err != nil {
					return errors.Wrapf(err, "while writing %s", nx.Key)
				}
			case dump.CreateMap:
				err = tx.CreateMap(nx.Key)
				if err != nil {
					return errors.Wrapf(err, "wile creating map %s", nx.Key)
				}
			default:
				return errors.Errorf("unsupported type %d", nx.Type)
			}

		}

		return nil
	})
}
