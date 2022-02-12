package testnet

import (
	"context"
	"encoding/hex"
	"github.com/tevino/abool"
	"math"
	"math/rand"
	"pandora-pay/addresses"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/plain_accounts"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/blockchain/transactions/transaction/transaction_simple"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload"
	"pandora-pay/config"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_stake"
	"pandora-pay/gui"
	"pandora-pay/helpers/generics"
	"pandora-pay/mempool"
	"pandora-pay/recovery"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_builder/wizard"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
	"time"
)

type Testnet struct {
	wallet     *wallet.Wallet
	mempool    *mempool.Mempool
	chain      *blockchain.Blockchain
	txsBuilder *txs_builder.TxsBuilder
	nodes      uint64
}

func (testnet *Testnet) testnetGetZetherRingConfiguration() *txs_builder.ZetherRingConfiguration {
	zetherRingConfiguration := &txs_builder.ZetherRingConfiguration{-1, -1}
	if config.LIGHT_COMPUTATIONS {
		zetherRingConfiguration.RingSize = int(math.Pow(2, float64(rand.Intn(2)+3)))
	}
	return zetherRingConfiguration
}

func (testnet *Testnet) testnetCreateClaimTx(recipientAddressWalletIndex int, amount uint64, ctx context.Context) (tx *transaction.Transaction, err error) {

	select {
	case <-ctx.Done():
		return
	default:
	}

	addr, err := testnet.wallet.GetWalletAddress(0, true)
	if err != nil {
		return
	}

	recipientAddr, err := testnet.wallet.GetWalletAddress(recipientAddressWalletIndex, true)
	if err != nil {
		return
	}

	var balance []byte
	if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {
		accs, err := accounts.NewAccountsCollection(reader).GetMap(config_coins.NATIVE_ASSET_FULL)
		if err != nil {
			return
		}
		acc, err := accs.GetAccount(recipientAddr.PublicKey)
		if err != nil || acc == nil {
			return
		}
		balance = acc.Balance.Amount.Serialize()
		return
	}); err != nil {
		return
	}

	if balance != nil {
		var decryptedBalance uint64
		if decryptedBalance, err = testnet.wallet.DecryptBalanceByPublicKey(recipientAddr.PublicKey, balance, config_coins.NATIVE_ASSET_FULL, false, 0, true, true, ctx, nil); err != nil {
			return
		}
		if decryptedBalance > config_coins.ConvertToUnitsUint64Forced(10000) {
			return
		}
	}

	senders := []string{""}
	recipients := []string{recipientAddr.AddressRegistrationEncoded}
	amounts, burn := []uint64{amount}, []uint64{0}
	sendAssets := [][]byte{config_coins.NATIVE_ASSET_FULL}
	data := []*wizard.WizardTransactionData{{[]byte{}, false}}
	fees := []*wizard.WizardZetherTransactionFee{{&wizard.WizardTransactionFee{0, 0, 0, true}, false, 0, 0}}

	if tx, err = testnet.txsBuilder.CreateZetherTx([]wizard.WizardZetherPayloadExtra{&wizard.WizardZetherPayloadExtraClaim{DelegatePrivateKey: addr.PrivateKey.Key}}, senders, sendAssets, amounts, recipients, burn, []*txs_builder.ZetherRingConfiguration{testnet.testnetGetZetherRingConfiguration()}, data, fees, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Claim Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).ChainHeight, tx.Bloom.Hash)
	return
}

func (testnet *Testnet) testnetCreateUnstakeTx(blockHeight uint64, amount uint64, ctx context.Context) (tx *transaction.Transaction, err error) {

	addr, err := testnet.wallet.GetWalletAddress(0, true)
	if err != nil {
		return
	}

	if tx, err = testnet.txsBuilder.CreateSimpleTx(addr.AddressEncoded, 0, &wizard.WizardTxSimpleExtraUnstake{Amount: amount}, &wizard.WizardTransactionData{nil, false}, &wizard.WizardTransactionFee{0, 0, 0, true}, false, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Unstake tx was created: " + hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) testnetCreateTransfersNewWallets(blockHeight uint64, ctx context.Context) (tx *transaction.Transaction, err error) {

	senders := []string{}
	recipients := []string{}
	amounts, burns := []uint64{}, []uint64{}
	sendAssets := [][]byte{}
	data := []*wizard.WizardTransactionData{}
	ringsConfigurations := []*txs_builder.ZetherRingConfiguration{}
	fees := []*wizard.WizardZetherTransactionFee{}
	payloadsExtra := []wizard.WizardZetherPayloadExtra{}

	for i := uint64(0); i < testnet.nodes; i++ {

		var addr *wallet_address.WalletAddress

		if addr, err = testnet.wallet.GetWalletAddress(1, true); err != nil {
			return
		}
		senders = append(senders, addr.AddressEncoded)

		if addr, err = testnet.wallet.GetWalletAddress(int(i+1), true); err != nil {
			return
		}

		asset := config_coins.NATIVE_ASSET_FULL

		recipients = append(recipients, addr.AddressRegistrationEncoded)
		amounts = append(amounts, config_stake.GetRequiredStake(blockHeight))
		sendAssets = append(sendAssets, asset)
		burns = append(burns, 0)

		ringsConfigurations = append(ringsConfigurations, testnet.testnetGetZetherRingConfiguration())

		data = append(data, &wizard.WizardTransactionData{[]byte{}, false})
		fees = append(fees, &wizard.WizardZetherTransactionFee{&wizard.WizardTransactionFee{0, 0, 0, true}, false, 0, 0})
		payloadsExtra = append(payloadsExtra, nil)
	}

	if tx, err = testnet.txsBuilder.CreateZetherTx(payloadsExtra, senders, sendAssets, amounts, recipients, burns, ringsConfigurations, data, fees, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).ChainHeight, tx.Bloom.Hash)
	return
}

func (testnet *Testnet) testnetCreateTransfers(senderAddressWalletIndex int, ctx context.Context) (tx *transaction.Transaction, err error) {

	select {
	case <-ctx.Done():
		return
	default:
	}

	senderAddr, err := testnet.wallet.GetWalletAddress(senderAddressWalletIndex, true)
	if err != nil {
		return
	}

	amount := uint64(rand.Int63n(6))
	burn := uint64(0)

	privateKey := addresses.GenerateNewPrivateKey()

	addr, err := privateKey.GenerateAddress(true, nil, 0, nil)
	if err != nil {
		return
	}

	recipient := addr.EncodeAddr()

	data := &wizard.WizardTransactionData{nil, false}
	fees := []*wizard.WizardZetherTransactionFee{{&wizard.WizardTransactionFee{0, 0, 0, true}, false, 0, 0}}

	if tx, err = testnet.txsBuilder.CreateZetherTx([]wizard.WizardZetherPayloadExtra{nil}, []string{senderAddr.AddressEncoded}, [][]byte{config_coins.NATIVE_ASSET_FULL}, []uint64{amount}, []string{recipient}, []uint64{burn}, []*txs_builder.ZetherRingConfiguration{testnet.testnetGetZetherRingConfiguration()}, []*wizard.WizardTransactionData{data}, fees, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).ChainHeight, tx.Bloom.Hash)
	return
}

func (testnet *Testnet) run() {

	updateChannel := testnet.chain.UpdateNewChain.AddListener()
	defer testnet.chain.UpdateNewChain.RemoveChannel(updateChannel)

	creatingTransactions := abool.New()

	for i := uint64(0); i < testnet.nodes; i++ {
		if uint64(testnet.wallet.GetAddressesCount()) <= i+1 {
			if _, err := testnet.wallet.AddNewAddress(true); err != nil {
				return
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {

		blockHeight, ok := <-updateChannel
		if !ok {
			return
		}

		syncTime := testnet.chain.Sync.GetSyncTime()

		recovery.SafeGo(func() {

			gui.GUI.Log("UpdateNewChain received! 1")
			defer gui.GUI.Log("UpdateNewChain received! DONE")

			if err := func() (err error) {

				if blockHeight == 20 {
					if _, err = testnet.testnetCreateUnstakeTx(blockHeight, testnet.nodes*config_stake.GetRequiredStake(blockHeight), ctx); err != nil {
						return
					}
				}
				if blockHeight == 100 {
					if _, err = testnet.testnetCreateTransfersNewWallets(blockHeight, ctx); err != nil {
						return
					}
				}

				if blockHeight >= 40 && syncTime != 0 {

					var addr *wallet_address.WalletAddress
					addr, _ = testnet.wallet.GetWalletAddress(0, true)

					var delegatedStakeAvailable, delegatedUnstakePending, unclaimed uint64

					var plainAcc *plain_account.PlainAccount

					gui.GUI.Log("UpdateNewChain received! 2")

					if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

						plainAccs := plain_accounts.NewPlainAccounts(reader)
						if plainAcc, err = plainAccs.GetPlainAccount(addr.PublicKey, blockHeight); err != nil {
							return
						}

						if plainAcc != nil {
							delegatedStakeAvailable = plainAcc.DelegatedStake.GetDelegatedStakeAvailable()
							delegatedUnstakePending, _ = plainAcc.DelegatedStake.ComputeDelegatedUnstakePending()
							unclaimed = plainAcc.Unclaimed
						}

						return
					}); err != nil {
						return
					}

					if plainAcc != nil {

						if creatingTransactions.IsNotSet() && syncTime != 0 {

							creatingTransactions.Set()
							defer creatingTransactions.UnSet()

							if unclaimed > config_coins.ConvertToUnitsUint64Forced(200) {

								unclaimed -= config_coins.ConvertToUnitsUint64Forced(30)

								if !testnet.mempool.ExistsTxZetherVersion(addr.PublicKey, transaction_zether_payload.SCRIPT_CLAIM) {
									testnet.testnetCreateClaimTx(1, unclaimed/5, ctx)
									testnet.testnetCreateClaimTx(2, unclaimed/5, ctx)
									testnet.testnetCreateClaimTx(3, unclaimed/5, ctx)
									testnet.testnetCreateClaimTx(4, unclaimed/5, ctx)
								}

							} else if delegatedStakeAvailable > 0 && unclaimed < delegatedStakeAvailable/4 && delegatedUnstakePending == 0 && delegatedStakeAvailable > config_coins.ConvertToUnitsUint64Forced(5000) && unclaimed < config_coins.ConvertToUnitsUint64Forced(2000) {
								if !testnet.mempool.ExistsTxSimpleVersion(addr.PublicKey, transaction_simple.SCRIPT_UNSTAKE) {
									if _, err = testnet.testnetCreateUnstakeTx(blockHeight, generics.Min(config_coins.ConvertToUnitsUint64Forced(1000), delegatedStakeAvailable/4), ctx); err != nil {
										return
									}
								}
							}

							time.Sleep(time.Millisecond * 500) //making sure the block got propagated
							for i := 2; i < 5; i++ {
								testnet.testnetCreateTransfers(i, ctx)
								time.Sleep(time.Millisecond * 5000)
							}
						}

					}

				}

				return
			}(); err != nil {
				gui.GUI.Error("Error creating testnet Tx", err)
				err = nil
			}

		})

	}

}

func TestnetInit(wallet *wallet.Wallet, mempool *mempool.Mempool, chain *blockchain.Blockchain, txsBuilder *txs_builder.TxsBuilder) (testnet *Testnet) {

	testnet = &Testnet{
		wallet:     wallet,
		mempool:    mempool,
		chain:      chain,
		txsBuilder: txsBuilder,
		nodes:      uint64(config.CPU_THREADS),
	}

	recovery.SafeGo(testnet.run)

	return
}
