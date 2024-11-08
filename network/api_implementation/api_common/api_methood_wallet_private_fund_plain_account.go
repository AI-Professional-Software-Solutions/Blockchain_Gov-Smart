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

type APIWalletPrivateFundPlainAccountRequest struct {
	Sender       string `json:"sender" msgpack:"sender"`
	PlainAccount string `json:"plainAccount" msgpack:"plainAccount"`
	Amount       uint64 `json:"amount" msgpack:"amount"`
	Propagate    bool   `json:"propagate" msgpack:"propagate"`
}

type APIWalletPrivateFundPlainAccountReply struct {
	Result bool                     `json:"result" msgpack:"result"`
	Tx     *transaction.Transaction `json:"tx" msgpack:"tx"`
}

func (api *APICommon) WalletPrivateFundPlainAccount(r *http.Request, args *APIWalletPrivateFundPlainAccountRequest, reply *APIWalletPrivateFundPlainAccountReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	extra := &wizard.WizardZetherPayloadExtraPlainAccountFund{}
	txData := &txs_builder.TxBuilderCreateZetherTxData{
		Payloads: []*txs_builder.TxBuilderCreateZetherTxPayload{{
			Extra: extra,
			Asset: config_coins.NATIVE_ASSET_FULL,
		}},
	}

	txData.Payloads[0].Sender = args.Sender

	var plainAccountAddress *addresses.Address
	if plainAccountAddress, err = addresses.DecodeAddr(args.PlainAccount); err != nil {
		return err
	}

	extra.PlainAccountPublicKey = plainAccountAddress.PublicKey
	txData.Payloads[0].Burn = args.Amount

	if reply.Tx, err = txs_builder.TxsBuilder.CreateZetherTx(txData, nil, args.Propagate, true, true, false, context.Background(), func(string) {}); err != nil {
		return
	}

	reply.Result = true

	return
}
