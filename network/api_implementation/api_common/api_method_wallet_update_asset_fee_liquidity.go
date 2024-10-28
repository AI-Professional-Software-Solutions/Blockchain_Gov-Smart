package api_common

import (
	"context"
	"errors"
	"net/http"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account/asset_fee_liquidity"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_builder/wizard"
)

type APIWalletUpdateAssetFeeLiquidityRequest struct {
	Sender       string `json:"sender" msgpack:"sender"`
	AssetId      []byte `json:"assetId" msgpack:"assetId"`
	Rate         uint64 `json:"rate" msgpack:"rate"`
	LeadingZeros byte   `json:"leadingZeros" msgpack:"leadingZeros"`
	Propagate    bool   `json:"propagate" msgpack:"propagate"`
	Collector    string `json:"collector" msgpack:"collector"`
}

type APIWalletUpdateAssetFeeLiquidityReply struct {
	Result bool                     `json:"result" msgpack:"result"`
	Tx     *transaction.Transaction `json:"tx" msgpack:"tx"`
}

func (api *APICommon) WalletUpdateAssetFeeLiquidity(r *http.Request, args *APIWalletUpdateAssetFeeLiquidityRequest, reply *APIWalletUpdateAssetFeeLiquidityReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	extra := &wizard.WizardTxSimpleExtraUpdateAssetFeeLiquidity{}
	txData := &txs_builder.TxBuilderCreateSimpleTx{
		Extra:      extra,
		FeeVersion: true,
	}

	txData.Sender = args.Sender

	if len(args.Collector) > 0 {
		var addr *addresses.Address
		if addr, err = addresses.DecodeAddr(args.Collector); err != nil {
			return err
		}
		extra.NewCollector = true
		extra.Collector = addr.PublicKey
	}

	liquidity := &asset_fee_liquidity.AssetFeeLiquidity{}
	liquidity.Asset = args.AssetId
	liquidity.Rate = args.Rate
	liquidity.LeadingZeros = byte(args.LeadingZeros)
	extra.Liquidities = append(extra.Liquidities, liquidity)

	if reply.Tx, err = txs_builder.TxsBuilder.CreateSimpleTx(txData, args.Propagate, true, true, false, context.Background(), func(string) {}); err != nil {
		return
	}

	reply.Result = true

	return
}
