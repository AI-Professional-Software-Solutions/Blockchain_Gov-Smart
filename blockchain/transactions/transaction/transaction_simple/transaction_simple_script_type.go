package transaction_simple

type ScriptType uint64

const (
	SCRIPT_UPDATE_ASSET_FEE_LIQUIDITY ScriptType = iota
	SCRIPT_RESOLUTION_CONDITIONAL_PAYMENT
	SCRIPT_NOTHING
)

func (t ScriptType) String() string {
	switch t {
	case SCRIPT_UPDATE_ASSET_FEE_LIQUIDITY:
		return "SCRIPT_UPDATE_ASSET_FEE_LIQUIDITY"
	case SCRIPT_RESOLUTION_CONDITIONAL_PAYMENT:
		return "SCRIPT_RESOLUTION_CONDITIONAL_PAYMENT"
	case SCRIPT_NOTHING:
		return "SCRIPT_NOTHING"
	default:
		return "Unknown ScriptType"
	}
}
