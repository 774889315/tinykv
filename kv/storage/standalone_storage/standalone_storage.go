package standalone_storage

import (
	"github.com/Connor1996/badger"
	"github.com/pingcap-incubator/tinykv/kv/config"
	"github.com/pingcap-incubator/tinykv/kv/storage"
	"github.com/pingcap-incubator/tinykv/kv/util/engine_util"
	"github.com/pingcap-incubator/tinykv/proto/pkg/kvrpcpb"
)

// StandAloneStorage is an implementation of `Storage` for a single-node TinyKV instance. It does not
// communicate with other nodes and all data is stored locally.
type StandAloneStorage struct {
	// Your Data Here (1).
	db *badger.DB
}

func NewStandAloneStorage(conf *config.Config) *StandAloneStorage {
	// Your Code Here (1).
	return &StandAloneStorage{
		db: engine_util.CreateDB("test", conf),
	}
}

func (s *StandAloneStorage) Start() error {
	// Your Code Here (1).
	return nil
}

func (s *StandAloneStorage) Stop() error {
	// Your Code Here (1).
	return nil
}

func (s *StandAloneStorage) Reader(ctx *kvrpcpb.Context) (storage.StorageReader, error) {
	// Your Code Here (1).
	return NewStandAloneStorageReader(s.db), nil
}

func (s *StandAloneStorage) Write(ctx *kvrpcpb.Context, batch []storage.Modify) error {
	// Your Code Here (1).
	txn := s.db.NewTransaction(true)
	var err error = nil
	switch batch[0].Data.(type) {
	case storage.Put:
		err = txn.Set(engine_util.KeyWithCF(batch[0].Cf(), batch[0].Key()), batch[0].Value())
	case storage.Delete:
		err = txn.Delete(engine_util.KeyWithCF(batch[0].Cf(), batch[0].Key()))
	}
	if err != nil {
		return err
	}
	return txn.Commit()
}

type StandAloneStorageReader struct {
	Txn *badger.Txn
}

func NewStandAloneStorageReader(db *badger.DB) *StandAloneStorageReader {
	txn := db.NewTransaction(false)
	return &StandAloneStorageReader {
		Txn: txn,
	}
}

func (sr *StandAloneStorageReader) GetCF(cf string, key []byte) ([]byte, error) {
	val, _ := engine_util.GetCFFromTxn(sr.Txn, cf, key)
	return val, nil
}

func (sr *StandAloneStorageReader) IterCF(cf string) engine_util.DBIterator {
	return engine_util.NewCFIterator(cf, sr.Txn)
}

func (sr *StandAloneStorageReader) Close() {
	sr.Txn.Commit()
}