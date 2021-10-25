package testnet

import (
	"context"
	"encoding/hex"
	"github.com/tevino/abool"
	"math/rand"
	"pandora-pay/addresses"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/plain_accounts"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/blockchain/transactions/transaction/transaction_simple"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload"
	"pandora-pay/config"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_stake"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/mempool"
	"pandora-pay/recovery"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/transactions_builder"
	"pandora-pay/transactions_builder/wizard"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
	"time"
)

type Testnet struct {
	wallet              *wallet.Wallet
	mempool             *mempool.Mempool
	chain               *blockchain.Blockchain
	transactionsBuilder *transactions_builder.TransactionsBuilder
	nodes               uint64
}

func (testnet *Testnet) testnetCreateClaimTx(dstAddressWalletIndex int, amount uint64, ctx context.Context) (tx *transaction.Transaction, err error) {

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	dstAddr, err := testnet.wallet.GetWalletAddress(dstAddressWalletIndex)
	if err != nil {
		return
	}

	from := []string{""}
	dsts := []string{dstAddr.AddressRegistrationEncoded}
	dstsAmounts, burn := []uint64{amount}, []uint64{0}
	dstsAssets := [][]byte{config_coins.NATIVE_ASSET_FULL}
	data := []*wizard.TransactionsWizardData{{[]byte{}, false}}
	fees := []*wizard.TransactionsWizardFee{{0, 0, 0, true}}

	var ring []string
	if ring, err = testnet.transactionsBuilder.CreateZetherRing(from[0], addr.AddressEncoded, dstsAssets[0], -1, -1); err != nil {
		return
	}
	ringMembers := [][]string{ring}

	if tx, err = testnet.transactionsBuilder.CreateZetherTx([]wizard.ZetherTransferPayloadExtra{&wizard.ZetherTransferPayloadExtraClaimStake{DelegatePrivateKey: addr.PrivateKey.Key}}, from, dstsAssets, dstsAmounts, dsts, burn, ringMembers, data, fees, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Claim Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).Height, tx.Bloom.Hash)
	return
}

func (testnet *Testnet) testnetCreateUnstakeTx(blockHeight uint64, amount uint64) (tx *transaction.Transaction, err error) {

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	if tx, err = testnet.transactionsBuilder.CreateUnstakeTx(addr.AddressEncoded, 0, amount, &wizard.TransactionsWizardData{nil, false}, &wizard.TransactionsWizardFee{0, 0, 0, true}, true, true, true, false, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Unstake tx was created: " + hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) testnetCreateTransfersNewWallets(blockHeight uint64, ctx context.Context) (tx *transaction.Transaction, err error) {

	from := []string{}
	dsts := []string{}
	dstsAmounts, burn := []uint64{}, []uint64{}
	dstsAssets := [][]byte{}
	data := []*wizard.TransactionsWizardData{}
	ringMembers := [][]string{}
	fees := []*wizard.TransactionsWizardFee{}
	payloadsExtra := []wizard.ZetherTransferPayloadExtra{}

	for i := uint64(0); i < testnet.nodes; i++ {

		var addr *wallet_address.WalletAddress

		if addr, err = testnet.wallet.GetWalletAddress(0); err != nil {
			return
		}
		from = append(from, addr.AddressEncoded)

		if addr, err = testnet.wallet.GetWalletAddress(int(i + 1)); err != nil {
			return
		}

		asset := config_coins.NATIVE_ASSET_FULL

		dsts = append(dsts, addr.AddressRegistrationEncoded)
		dstsAmounts = append(dstsAmounts, config_stake.GetRequiredStake(blockHeight))
		dstsAssets = append(dstsAssets, asset)
		burn = append(burn, 0)

		var ring []string
		if ring, err = testnet.transactionsBuilder.CreateZetherRing(from[i], addr.AddressEncoded, asset, -1, -1); err != nil {
			return
		}
		ringMembers = append(ringMembers, ring)

		data = append(data, &wizard.TransactionsWizardData{[]byte{}, false})
		fees = append(fees, &wizard.TransactionsWizardFee{0, 0, 0, true})
		payloadsExtra = append(payloadsExtra, nil)
	}

	if tx, err = testnet.transactionsBuilder.CreateZetherTx(payloadsExtra, from, dstsAssets, dstsAmounts, dsts, burn, ringMembers, data, fees, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).Height, tx.Bloom.Hash)
	return
}

func (testnet *Testnet) testnetCreateTransfers(srcAddressWalletIndex int, ctx context.Context) (tx *transaction.Transaction, err error) {

	srcAddr, err := testnet.wallet.GetWalletAddress(srcAddressWalletIndex)
	if err != nil {
		return
	}

	amount := uint64(rand.Int63n(6))
	burn := uint64(0)

	privateKey := addresses.GenerateNewPrivateKey()

	addr, err := privateKey.GenerateAddress(true, 0, helpers.EmptyBytes(0))
	if err != nil {
		return
	}

	dst := addr.EncodeAddr()

	data := &wizard.TransactionsWizardData{nil, false}
	fee := &wizard.TransactionsWizardFee{0, 0, 0, true}

	ringMembers, err := testnet.transactionsBuilder.CreateZetherRing(srcAddr.AddressEncoded, dst, config_coins.NATIVE_ASSET_FULL, -1, -1)
	if err != nil {
		return
	}

	if tx, err = testnet.transactionsBuilder.CreateZetherTx([]wizard.ZetherTransferPayloadExtra{nil}, []string{srcAddr.AddressEncoded}, [][]byte{config_coins.NATIVE_ASSET_FULL}, []uint64{amount}, []string{dst}, []uint64{burn}, [][]string{ringMembers}, []*wizard.TransactionsWizardData{data}, []*wizard.TransactionsWizardFee{fee}, true, true, true, false, ctx, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_zether.TransactionZether).Height, tx.Bloom.Hash)
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

	//var ctx context.Context
	//var cancel context.CancelFunc
	ctx2, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {

		blockHeightReceived, ok := <-updateChannel
		if !ok {
			return
		}

		blockHeight := blockHeightReceived.(uint64)
		syncTime := testnet.chain.Sync.GetSyncTime()

		recovery.SafeGo(func() {

			gui.GUI.Log("UpdateNewChain received! 1")
			defer gui.GUI.Log("UpdateNewChain received! DONE")

			err := func() (err error) {

				if blockHeight == 20 {
					if _, err = testnet.testnetCreateUnstakeTx(blockHeight, testnet.nodes*config_stake.GetRequiredStake(blockHeight)); err != nil {
						return
					}
				}
				if blockHeight == 100 {
					if _, err = testnet.testnetCreateTransfersNewWallets(blockHeight, ctx2); err != nil {
						return
					}
				}

				if blockHeight >= 40 && syncTime != 0 {

					var addr *wallet_address.WalletAddress
					addr, _ = testnet.wallet.GetWalletAddress(0)

					publicKey := addr.PublicKey

					var delegatedStakeAvailable, delegatedUnstakePending, unclaimed uint64
					var balanceHomo *crypto.ElGamal

					var acc *account.Account
					var plainAcc *plain_account.PlainAccount

					gui.GUI.Log("UpdateNewChain received! 2")

					if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

						accsCollection := accounts.NewAccountsCollection(reader)

						accs, err := accsCollection.GetMap(config_coins.NATIVE_ASSET_FULL)
						if err != nil {
							return
						}
						if acc, err = accs.GetAccount(publicKey); err != nil {
							return
						}

						plainAccs := plain_accounts.NewPlainAccounts(reader)
						if plainAcc, err = plainAccs.GetPlainAccount(publicKey, blockHeight); err != nil {
							return
						}

						if acc != nil {
							balanceHomo = acc.GetBalance()
						}

						if plainAcc != nil {
							delegatedStakeAvailable = plainAcc.GetDelegatedStakeAvailable()
							delegatedUnstakePending, _ = plainAcc.ComputeDelegatedUnstakePending()
							unclaimed = plainAcc.Unclaimed
						}

						return
					}); err != nil {
						return
					}

					if acc != nil || plainAcc != nil {

						var balance uint64
						if acc != nil {
							if balance, err = testnet.wallet.DecodeBalanceByPublicKey(publicKey, balanceHomo, config_coins.NATIVE_ASSET_FULL, true, true, ctx2, func(string) {}); err != nil {
								return
							}
						}

						if creatingTransactions.IsNotSet() {

							creatingTransactions.Set()
							defer creatingTransactions.UnSet()

							if unclaimed > config_coins.ConvertToUnitsUint64Forced(10) {

								if !testnet.mempool.ExistsTxZetherVersion(addr.PublicKey, transaction_zether_payload.SCRIPT_CLAIM_STAKE) {
									testnet.testnetCreateClaimTx(0, unclaimed/4, ctx2)
									testnet.testnetCreateClaimTx(1, unclaimed/4, ctx2)
									testnet.testnetCreateClaimTx(2, unclaimed/4, ctx2)
									testnet.testnetCreateClaimTx(3, unclaimed/4-config_coins.ConvertToUnitsUint64Forced(10), ctx2)
								}

							} else if delegatedStakeAvailable > 0 && balance < delegatedStakeAvailable/4 && delegatedUnstakePending == 0 {
								if !testnet.mempool.ExistsTxSimpleVersion(addr.PublicKey, transaction_simple.SCRIPT_UNSTAKE) {
									if _, err = testnet.testnetCreateUnstakeTx(blockHeight, delegatedStakeAvailable/2-balance); err != nil {
										return
									}
								}
							} else {

								for i := 1; i < 4; i++ {
									testnet.testnetCreateTransfers(i, ctx2)
									time.Sleep(time.Millisecond * 50)
								}

							}
						}

					}

				}

				return
			}()

			if err != nil {
				gui.GUI.Error("Error creating testnet Tx", err)
				err = nil
			}

		})

	}
}

func TestnetInit(wallet *wallet.Wallet, mempool *mempool.Mempool, chain *blockchain.Blockchain, transactionsBuilder *transactions_builder.TransactionsBuilder) (testnet *Testnet) {

	testnet = &Testnet{
		wallet:              wallet,
		mempool:             mempool,
		chain:               chain,
		transactionsBuilder: transactionsBuilder,
		nodes:               uint64(config.CPU_THREADS),
	}

	recovery.SafeGo(testnet.run)

	return
}
