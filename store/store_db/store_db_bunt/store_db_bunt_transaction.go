package store_db_bunt

import (
	buntdb "github.com/tidwall/buntdb"
	"pandora-pay/helpers"
	"pandora-pay/store/store_db/store_db_interface"
)

type StoreDBBuntTransaction struct {
	store_db_interface.StoreDBTransactionInterface
	buntTx *buntdb.Tx
	write  bool
}

func (tx *StoreDBBuntTransaction) IsWritable() bool {
	return tx.write
}

func (tx *StoreDBBuntTransaction) Put(key string, value []byte) {
	if _, _, err := tx.buntTx.Set(key, string(value), nil); err != nil {
		panic(err)
	}
}

func (tx *StoreDBBuntTransaction) Get(key string) (out []byte) {
	data, err := tx.buntTx.Get(key, false)
	if err == nil {
		out = []byte(data)
	}
	return
}

func (tx *StoreDBBuntTransaction) Exists(key string) bool {
	_, err := tx.buntTx.Get(key, false)
	if err == nil {
		return true
	}
	return false
}

func (tx *StoreDBBuntTransaction) GetClone(key string) []byte {
	//TODO: check if cloneBytes is necessary for BuntDB
	return helpers.CloneBytes(tx.Get(key))
}

func (tx *StoreDBBuntTransaction) PutClone(key string, value []byte) {
	tx.Put(key, helpers.CloneBytes(value))
}

func (tx *StoreDBBuntTransaction) Delete(key string) {
	_, err := tx.buntTx.Delete(key)
	if err != buntdb.ErrNotFound {
		panic(err)
	}
}
