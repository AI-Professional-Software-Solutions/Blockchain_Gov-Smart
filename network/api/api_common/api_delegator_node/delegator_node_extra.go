package api_delegator_node

import (
	"bytes"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/config/config_nodes"
	"pandora-pay/gui"
	"pandora-pay/recovery"
	"pandora-pay/wallet/wallet_address"
	"sync/atomic"
	"time"
)

func (api *DelegatorNode) execute() {
	recovery.SafeGo(func() {

		updateNewChainUpdateListener := api.chain.UpdateNewChain.AddListener()
		defer api.chain.UpdateNewChain.RemoveChannel(updateNewChainUpdateListener)

		for {

			chainHeight, ok := <-updateNewChainUpdateListener
			if !ok {
				return
			}

			atomic.StoreUint64(&api.chainHeight, chainHeight)
		}
	})

	recovery.SafeGo(func() {

		lastHeight := uint64(0)

		for {
			chainHeight := atomic.LoadUint64(&api.chainHeight)
			if lastHeight != chainHeight {
				lastHeight = chainHeight

				api.pendingDelegatesStakesChanges.Range(func(key string, pendingDelegateStakeChange *PendingDelegateStakeChange) bool {
					if chainHeight >= pendingDelegateStakeChange.blockHeight+100 {
						api.pendingDelegatesStakesChanges.Delete(key)
					}
					return true
				})
			}

			time.Sleep(10 * time.Second)
		}
	})

	api.updateAccountsChanges()

}

func (api *DelegatorNode) updateAccountsChanges() {

	recovery.SafeGo(func() {

		updatePlainAccountsCn := api.chain.UpdatePlainAccounts.AddListener()
		defer api.chain.UpdatePlainAccounts.RemoveChannel(updatePlainAccountsCn)

		for {

			plainAccs, ok := <-updatePlainAccountsCn
			if !ok {
				return
			}

			for k, v := range plainAccs.HashMap.Committed {

				pendingDelegatingStakeChange, loaded := api.pendingDelegatesStakesChanges.Load(k)
				if loaded {

					if v.Stored == "update" {
						plainAcc := v.Element.(*plain_account.PlainAccount)
						if plainAcc.DelegatedStake.HasDelegatedStake() && bytes.Equal(plainAcc.DelegatedStake.DelegatedStakePublicKey, pendingDelegatingStakeChange.delegateStakingPublicKey) {

							if plainAcc.DelegatedStake.DelegatedStakeFee < config_nodes.DELEGATOR_FEE {
								continue
							}

							if plainAcc.DelegatedStake.DelegatedStakeFee > 0 && len(config_nodes.DELEGATOR_REWARD_COLLECTOR_PUBLIC_KEY) == 0 {
								gui.GUI.Error("You need to set the DELEGATOR_REWARD_COLLECTOR_PUBLIC_KEY")
							}

							addr, err := addresses.CreateAddr(pendingDelegatingStakeChange.publicKey, nil, nil, 0, nil)
							if err != nil {
								continue
							}

							_ = api.wallet.AddDelegateStakeAddress(&wallet_address.WalletAddress{
								wallet_address.VERSION_NORMAL,
								"Delegated Stake",
								0,
								false,
								nil,
								nil,
								pendingDelegatingStakeChange.publicKey,
								make(map[string]*wallet_address.WalletAddressDecryptedBalance),
								addr.EncodeAddr(),
								"",
								&wallet_address.WalletAddressDelegatedStake{
									&addresses.PrivateKey{Key: pendingDelegatingStakeChange.delegateStakingPrivateKey.Key},
									pendingDelegatingStakeChange.delegateStakingPublicKey,
									0,
								},
							}, true)
						}
					}

				}
			}
		}
	})

}
