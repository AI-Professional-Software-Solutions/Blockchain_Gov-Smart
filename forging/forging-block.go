package forging

import (
	"encoding/binary"
	"pandora-pay/blockchain/block/difficulty"
	"pandora-pay/config"
	"pandora-pay/config/stake"
	"pandora-pay/cryptography"
	"sync/atomic"
	"time"
	"unsafe"
)

//inside a thread
func forge(forging *Forging, workPointer unsafe.Pointer, work *ForgingWork, wallets []*ForgingWalletAddress) {

	buf := make([]byte, binary.MaxVarintLen64)

	defer forging.wg.Done()

	height := work.blkComplete.Block.Height
	serialized := work.blkComplete.Block.SerializeForForging()
	n := binary.PutUvarint(buf, work.blkComplete.Block.Timestamp)

	serialized = serialized[:len(serialized)-n-20]
	timestamp := work.blkComplete.Block.Timestamp + 1

	for atomic.LoadPointer(&forging.work) == workPointer {

		if timestamp > uint64(time.Now().Unix())+config.NETWORK_TIMESTAMP_DRIFT_MAX {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		//forge with my wallets
		for _, address := range wallets {

			var stakingAmount uint64
			if address.account != nil {
				stakingAmount = address.account.GetDelegatedStakeAvailable(height)
			}

			if stakingAmount >= stake.GetRequiredStake(height) {

				if atomic.LoadPointer(&forging.work) != workPointer {
					break
				}

				n = binary.PutUvarint(buf, timestamp)
				serialized = append(serialized, buf[:n]...)
				serialized = append(serialized, address.publicKeyHash[:]...)
				kernelHash := cryptography.SHA3Hash(serialized)

				if height > 0 {
					kernelHash = cryptography.ComputeKernelHash(kernelHash, stakingAmount)
				}

				if difficulty.CheckKernelHashBig(kernelHash, work.target) {

					forging.foundSolution(address, timestamp, work)
					return

				} else {
					// for debugging only
					// gui.Log(hex.EncodeToString(kernelHash[:]))
				}

				serialized = serialized[:len(serialized)-n-20]

			}

		}
		timestamp += 1

	}

}