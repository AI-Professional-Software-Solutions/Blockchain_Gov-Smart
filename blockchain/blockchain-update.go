package blockchain

import (
	"encoding/hex"
	"fmt"
	"pandora-pay/blockchain/accounts"
	block_complete "pandora-pay/blockchain/block-complete"
	"pandora-pay/blockchain/tokens"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"sync"
	"sync/atomic"
	"time"
)

type BlockchainUpdate struct {
	err              error
	newChainData     *BlockchainData
	accs             *accounts.Accounts
	toks             *tokens.Tokens
	removedTxs       [][]byte
	insertedBlocks   []*block_complete.BlockComplete
	insertedTxHashes [][]byte
}

type BlockchainUpdatesQueue struct {
	updates        atomic.Value //[]*BlockchainUpdate
	updatesWriting *sync.Mutex
	chain          *Blockchain
}

func (queue *BlockchainUpdatesQueue) isLastSuccess(updates []*BlockchainUpdate) bool {
	for _, update := range updates {
		if update.err == nil {
			return false
		}
	}

	return true
}

func (queue *BlockchainUpdatesQueue) processUpdate(update *BlockchainUpdate, updates []*BlockchainUpdate) {

	if update.err != nil {
		if len(updates) == 0 && queue.isLastSuccess(updates) {
			queue.chain.createNextBlockForForging()
		}
		return
	}

	var err error

	gui.GUI.Warning("-------------------------------------------")
	gui.GUI.Warning(fmt.Sprintf("Included blocks %d | TXs: %d | Hash %s", len(update.insertedBlocks), len(update.insertedTxHashes), hex.EncodeToString(update.newChainData.Hash)))
	gui.GUI.Warning(update.newChainData.Height, hex.EncodeToString(update.newChainData.Hash), update.newChainData.Target.Text(10), update.newChainData.BigTotalDifficulty.Text(10))
	gui.GUI.Warning("-------------------------------------------")
	update.newChainData.updateChainInfo()

	//accs will only be read only
	if err = queue.chain.forging.Wallet.UpdateAccountsChanges(update.accs); err != nil {
		gui.GUI.Error("Error updating balance changes", err)
	}

	if err = queue.chain.wallet.UpdateAccountsChanges(update.accs); err != nil {
		gui.GUI.Error("Error updating balance changes", err)
	}

	if queue.isLastSuccess(updates) {
		if err = queue.chain.forging.Wallet.ProcessUpdates(); err != nil {
			gui.GUI.Error("Error Processing Updates", err)
		}
		//update work for mem pool
		queue.chain.mempool.UpdateWork(update.newChainData.Hash, update.newChainData.Height)

		//create next block and the workers will be automatically reset
		queue.chain.createNextBlockForForging()
	}

	for _, txData := range update.removedTxs {
		tx := &transaction.Transaction{}
		if err = tx.Deserialize(helpers.NewBufferReader(txData)); err != nil {
			return
		}
		if err = tx.BloomExtraNow(true); err != nil {
			return
		}
		if _, err = queue.chain.mempool.AddTxToMemPool(tx, update.newChainData.Height, false); err != nil {
			return
		}
	}

	queue.chain.mempool.DeleteTxs(update.insertedTxHashes)

	newSyncTime, result := queue.chain.Sync.addBlocksChanged(uint32(len(update.insertedBlocks)), false)

	if queue.isLastSuccess(updates) {

		queue.chain.UpdateMulticast.Broadcast(update.newChainData.Height)

		queue.chain.UpdateNewChainMulticast.Broadcast(update.newChainData)

		if result {
			queue.chain.Sync.UpdateSyncMulticast.Broadcast(newSyncTime)
		}
	}

}

func (queue *BlockchainUpdatesQueue) processQueue() {
	go func() {
		for {
			updates := queue.updates.Load().([]*BlockchainUpdate)
			if len(updates) > 0 {
				queue.updatesWriting.Lock()
				updates = queue.updates.Load().([]*BlockchainUpdate)
				update := updates[0]
				updates = append(updates[1:])
				queue.updates.Store(updates)
				queue.updatesWriting.Unlock()

				queue.processUpdate(update, updates)
				if len(updates) > 0 {
					continue
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
}