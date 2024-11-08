package api_common

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/assets/asset"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload/transaction_zether_payload_extra"
	"pandora-pay/config/config_coins"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_builder/wizard"
)

type APIWalletPrivateCreateAssetRequest struct {
	Sender    string       `json:"sender" msgpack:"sender"`
	Asset     *asset.Asset `json:"asset" msgpack:"asset"`
	Propagate bool         `json:"propagate" msgpack:"propagate"`
}

type APIWalletPrivateCreateAssetReply struct {
	Result                      bool                     `json:"result" msgpack:"result"`
	Tx                          *transaction.Transaction `json:"tx" msgpack:"tx"`
	AssetUpdatePrivateKey       []byte                   `json:"assetUpdatePrivateKey" msgpack:"assetUpdatePrivateKey"`
	AssetSupplyUpdatePrivateKey []byte                   `json:"assetSupplyUpdatePrivateKey" msgpack:"assetSupplyUpdatePrivateKey"`
	AssetId                     []byte                   `json:"assetId" msgpack:"assetId"`
}

func (api *APICommon) WalletPrivateCreateAsset(r *http.Request, args *APIWalletPrivateCreateAssetRequest, reply *APIWalletPrivateCreateAssetReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	extra := &wizard.WizardZetherPayloadExtraAssetCreate{}
	txData := &txs_builder.TxBuilderCreateZetherTxData{
		Payloads: []*txs_builder.TxBuilderCreateZetherTxPayload{{
			Extra: extra,
			Asset: config_coins.NATIVE_ASSET_FULL,
		}},
	}

	txData.Payloads[0].Sender = args.Sender
	extra.Asset = args.Asset
	if len(extra.Asset.PublicKeyHash) == 0 {
		extra.Asset.PublicKeyHash = helpers.RandomBytes(cryptography.PublicKeyHashSize)
	}

	extra.Asset.Identification = extra.Asset.Ticker + "-" + hex.EncodeToString(extra.Asset.PublicKeyHash[:3])

	if err = extra.Asset.Validate(); err != nil {
		return
	}

	if len(args.Asset.UpdatePublicKey) == 0 {
		updatePrivKey := addresses.GenerateNewPrivateKey()
		extra.Asset.UpdatePublicKey = updatePrivKey.GeneratePublicKey()
		reply.AssetUpdatePrivateKey = updatePrivKey.Key
	}
	if len(args.Asset.SupplyPublicKey) == 0 {
		supplyPrivKey := addresses.GenerateNewPrivateKey()
		extra.Asset.SupplyPublicKey = supplyPrivKey.GeneratePublicKey()
		reply.AssetSupplyUpdatePrivateKey = supplyPrivKey.Key
	}

	if reply.Tx, err = txs_builder.TxsBuilder.CreateZetherTx(txData, nil, args.Propagate, true, true, false, context.Background(), func(string) {}); err != nil {
		return
	}

	assetId := reply.Tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).Payloads[0].Extra.(*transaction_zether_payload_extra.TransactionZetherPayloadExtraAssetCreate).GetAssetId(reply.Tx.Bloom.Hash, 0)
	reply.AssetId = assetId

	reply.Result = true

	return
}
