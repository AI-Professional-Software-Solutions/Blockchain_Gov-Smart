package account

import (
	"bytes"
	"errors"
	"math/big"
	"pandora-pay/blockchain/accounts/account/dpos"
	"pandora-pay/config"
	"pandora-pay/cryptography"
	"pandora-pay/cryptography/cryptolib"
	"pandora-pay/helpers"
)

type Account struct {
	helpers.SerializableInterface `json:"-"`
	PublicKey                     []byte                `json:"-"`
	Version                       uint64                `json:"version"`
	Nonce                         uint64                `json:"nonce"`
	Balances                      []*Balance            `json:"balances"`
	BalancesHomo                  []*BalanceHomomorphic `json:"balancesHomo"`
	DelegatedStakeVersion         uint64                `json:"delegatedStakeVersion"`
	DelegatedStake                *dpos.DelegatedStake  `json:"delegatedStake"`
}

func (account *Account) Validate() error {
	if account.Version != 0 {
		return errors.New("Version is invalid")
	}
	if account.DelegatedStakeVersion > 1 {
		return errors.New("Invalid DelegatedStakeVersion version")
	}
	return nil
}

func (account *Account) HasDelegatedStake() bool {
	return account.DelegatedStakeVersion == 1
}

func (account *Account) IncrementNonce(sign bool) error {
	return helpers.SafeUint64Update(sign, &account.Nonce, 1)
}

//todo remove
func (account *Account) AddBalance(sign bool, amount uint64, tok []byte) (err error) {

	if amount == 0 {
		return
	}

	var foundBalance *Balance
	var foundBalanceIndex int

	for i, balance := range account.Balances {
		if bytes.Equal(balance.Token, tok) {
			foundBalance = balance
			foundBalanceIndex = i
			break
		}
	}

	if sign {
		if foundBalance == nil {
			foundBalance = &Balance{
				Token: tok,
			}
			account.Balances = append(account.Balances, foundBalance)
		}
		if err = helpers.SafeUint64Add(&foundBalance.Amount, amount); err != nil {
			return
		}
	} else {

		if foundBalance == nil {
			return errors.New("Balance doesn't exist or would become negative")
		}
		if err = helpers.SafeUint64Sub(&foundBalance.Amount, amount); err != nil {
			return
		}

		if foundBalance.Amount == 0 {
			//fast removal
			account.Balances[foundBalanceIndex] = account.Balances[len(account.Balances)-1]
			account.Balances = account.Balances[:len(account.Balances)-1]
		}

	}

	return
}

//todo remove
func (account *Account) GetAvailableBalance(token []byte) (result uint64) {
	for _, balance := range account.Balances {
		if bytes.Equal(balance.Token, token) {
			return balance.Amount
		}
	}
	return 0
}

func (account *Account) AddBalanceHomoUint(amount uint64, tok []byte) (err error) {

	var foundBalance *BalanceHomomorphic

	for _, balance := range account.BalancesHomo {
		if bytes.Equal(balance.Token, tok) {
			foundBalance = balance
			break
		}
	}

	if foundBalance == nil {
		var acckey cryptolib.Point
		if err := acckey.DecodeCompressed(account.PublicKey); err != nil {
			panic(err)
		}
		foundBalance = &BalanceHomomorphic{cryptolib.ConstructElGamal(acckey.G1(), cryptolib.ElGamal_BASE_G), tok}
		account.BalancesHomo = append(account.BalancesHomo, foundBalance)
	}

	foundBalance.Amount = foundBalance.Amount.Plus(new(big.Int).SetUint64(amount))

	return
}

func (account *Account) AddBalanceHomo(encryptedAmount []byte, tok []byte) (err error) {
	panic("not implemented")
}

func (account *Account) RefreshDelegatedStake(blockHeight uint64) (err error) {
	if account.DelegatedStakeVersion == 0 {
		return
	}

	for i := len(account.DelegatedStake.StakesPending) - 1; i >= 0; i-- {
		stakePending := account.DelegatedStake.StakesPending[i]
		if stakePending.ActivationHeight <= blockHeight {

			if stakePending.PendingType == dpos.DelegatedStakePendingStake {
				if err = helpers.SafeUint64Add(&account.DelegatedStake.StakeAvailable, stakePending.PendingAmount); err != nil {
					return
				}
			} else {
				if err = account.AddBalanceHomoUint(stakePending.PendingAmount, config.NATIVE_TOKEN); err != nil {
					return
				}
			}
			account.DelegatedStake.StakesPending = append(account.DelegatedStake.StakesPending[:i], account.DelegatedStake.StakesPending[i+1:]...)
		}
	}

	if account.DelegatedStake.IsDelegatedStakeEmpty() {
		account.DelegatedStakeVersion = 0
		account.DelegatedStake = nil
	}
	return
}

func (account *Account) GetDelegatedStakeAvailable() uint64 {
	if account.DelegatedStakeVersion == 0 {
		return 0
	}
	return account.DelegatedStake.GetDelegatedStakeAvailable()
}

func (account *Account) ComputeDelegatedStakeAvailable(chainHeight uint64) (uint64, error) {
	if account.DelegatedStakeVersion == 0 {
		return 0, nil
	}
	return account.DelegatedStake.ComputeDelegatedStakeAvailable(chainHeight)
}

func (account *Account) ComputeDelegatedUnstakePending() (uint64, error) {
	if account.DelegatedStakeVersion == 0 {
		return 0, nil
	}
	return account.DelegatedStake.ComputeDelegatedUnstakePending()
}

func (account *Account) GetBalanceHomo(token []byte) (result *cryptolib.ElGamal) {
	for _, balance := range account.BalancesHomo {
		if bytes.Equal(balance.Token, token) {
			return balance.Amount
		}
	}
	return nil
}

func (account *Account) Serialize(writer *helpers.BufferWriter) {

	writer.WriteUvarint(account.Version)
	writer.WriteUvarint(account.Nonce)

	writer.WriteUvarint16(uint16(len(account.Balances)))
	for _, balance := range account.Balances {
		balance.Serialize(writer)
	}

	writer.WriteUvarint16(uint16(len(account.BalancesHomo)))
	for _, balanceHomo := range account.BalancesHomo {
		balanceHomo.Serialize(writer)
	}

	writer.WriteUvarint(account.DelegatedStakeVersion)
	if account.DelegatedStakeVersion == 1 {
		account.DelegatedStake.Serialize(writer)
	}

}

func (account *Account) SerializeToBytes() []byte {
	writer := helpers.NewBufferWriter()
	account.Serialize(writer)
	return writer.Bytes()
}

func (account *Account) CreateDelegatedStake(amount uint64, delegatedStakePublicKey []byte, delegatedStakeFee uint16) error {
	if account.HasDelegatedStake() {
		return errors.New("It is already delegated")
	}
	if delegatedStakePublicKey == nil || len(delegatedStakePublicKey) != cryptography.PublicKeySize {
		return errors.New("delegatedStakePublicKey is Invalid")
	}
	account.DelegatedStakeVersion = 1
	account.DelegatedStake = &dpos.DelegatedStake{
		StakeAvailable:     amount,
		StakesPending:      []*dpos.DelegatedStakePending{},
		DelegatedPublicKey: delegatedStakePublicKey,
		DelegatedStakeFee:  delegatedStakeFee,
	}

	return nil
}

func (account *Account) Deserialize(reader *helpers.BufferReader) (err error) {

	if account.Version, err = reader.ReadUvarint(); err != nil {
		return
	}
	if account.Nonce, err = reader.ReadUvarint(); err != nil {
		return
	}

	var n uint16
	if n, err = reader.ReadUvarint16(); err != nil {
		return
	}
	account.Balances = make([]*Balance, n)
	for i := uint16(0); i < n; i++ {
		var balance = new(Balance)
		if err = balance.Deserialize(reader); err != nil {
			return
		}
		account.Balances[i] = balance
	}

	if n, err = reader.ReadUvarint16(); err != nil {
		return
	}
	account.BalancesHomo = make([]*BalanceHomomorphic, n)
	for i := uint16(0); i < n; i++ {
		var balance = new(BalanceHomomorphic)
		if err = balance.Deserialize(reader); err != nil {
			return
		}
		account.BalancesHomo[i] = balance
	}

	if account.DelegatedStakeVersion, err = reader.ReadUvarint(); err != nil {
		return
	}
	if account.DelegatedStakeVersion == 1 {
		account.DelegatedStake = new(dpos.DelegatedStake)
		if err = account.DelegatedStake.Deserialize(reader); err != nil {
			return
		}
	}

	return
}
