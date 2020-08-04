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
}

var Engine *badger.DB
var Txn *badger.Txn

func NewStandAloneStorage(conf *config.Config) *StandAloneStorage {
	// Your Code Here (1).
	Engine = engine_util.CreateDB("test", conf)
	Txn = Engine.NewTransaction(true)
	return nil
}

func (s *StandAloneStorage) Start() error {
	// Your Code Here (1).
	return nil
}

func (s *StandAloneStorage) Stop() error {
	// Your Code Here (1).
	Engine.Close()
	return nil
}

func (s *StandAloneStorage) Reader(ctx *kvrpcpb.Context) (storage.StorageReader, error) {
	// Your Code Here (1).
	return s, nil
}

func (s *StandAloneStorage) Write(ctx *kvrpcpb.Context, batch []storage.Modify) error {
	// Your Code Here (1).
	Txn.Set(engine_util.KeyWithCF(batch[0].Cf(), batch[0].Key()), batch[0].Value())
	return nil
}

func (s *StandAloneStorage) GetCF(cf string, key []byte) ([]byte, error) {
	val, _ := engine_util.GetCFFromTxn(Txn, cf, key)
	return val, nil
}

func (s *StandAloneStorage) IterCF(cf string) engine_util.DBIterator {
	panic("implement me")
}

func (s *StandAloneStorage) Close() {
	panic("implement me")
}