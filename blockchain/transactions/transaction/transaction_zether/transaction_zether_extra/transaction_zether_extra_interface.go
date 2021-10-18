package transaction_zether_extra

import (
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/transactions/transaction/transaction_data"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload"
	"pandora-pay/helpers"
)

type TransactionZetherExtraInterface interface {
	BeforeIncludeTransaction(txRegistrations *transaction_data.TransactionDataTransactions, payloads []*transaction_zether_payload.TransactionZetherPayload, publicKeyListByCounter [][]byte, blockHeight uint64, dataStorage *data_storage.DataStorage) error
	IncludeTransaction(txRegistrations *transaction_data.TransactionDataTransactions, payloads []*transaction_zether_payload.TransactionZetherPayload, publicKeyListByCounter [][]byte, blockHeight uint64, dataStorage *data_storage.DataStorage) error
	Serialize(w *helpers.BufferWriter, inclSignature bool)
	Deserialize(r *helpers.BufferReader) error
	Validate(txRegistrations *transaction_data.TransactionDataTransactions, payloads []*transaction_zether_payload.TransactionZetherPayload) error
}
