package api_common

import (
	"context"
	"errors"
	"net/http"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/config/config_coins"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_builder/wizard"
)

type APIWalletPrivateAssetSupplyIncreaseRequest struct {
	Sender                string `json:"sender" msgpack:"sender"`
	AssetRecipient        string `json:"assetRecipient" msgpack:"assetRecipient"`
	AssetId               []byte `json:"assetId" msgpack:"assetId"`
	AssetSupplyPrivateKey []byte `json:"assetSupplyPrivateKey" msgpack:"assetSupplyPrivateKey"`
	AssetSupply           uint64 `json:"assetSupply" msgpack:"assetSupply"`
	Propagate             bool   `json:"propagate" msgpack:"propagate"`
}

type APIWalletPrivateAssetSupplyIncreaseReply struct {
	Result bool                     `json:"result" msgpack:"result"`
	Tx     *transaction.Transaction `json:"tx" msgpack:"tx"`
}

func (api *APICommon) WalletPrivateAssetSupplyIncrease(r *http.Request, args *APIWalletPrivateAssetSupplyIncreaseRequest, reply *APIWalletPrivateAssetSupplyIncreaseReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	extra := &wizard.WizardZetherPayloadExtraAssetSupplyIncrease{}
	txData := &txs_builder.TxBuilderCreateZetherTxData{
		Payloads: []*txs_builder.TxBuilderCreateZetherTxPayload{{
			Extra: extra,
			Asset: config_coins.NATIVE_ASSET_FULL,
		}},
	}

	txData.Payloads[0].Sender = args.Sender

	extra.AssetId = args.AssetId
	extra.AssetSupplyPrivateKey = args.AssetSupplyPrivateKey
	extra.Value = args.AssetSupply

	addr, err := addresses.DecodeAddr(args.AssetRecipient)
	if err != nil {
		return err
	}

	extra.ReceiverPublicKey = addr.PublicKey

	if reply.Tx, err = txs_builder.TxsBuilder.CreateZetherTx(txData, nil, args.Propagate, true, true, false, context.Background(), func(string) {}); err != nil {
		return
	}

	reply.Result = true

	return
}
