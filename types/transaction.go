package types

import (
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/0xPolygon/polygon-edge/helper/keccak"
)

const (
	// StateTransactionGasLimit is arbitrary default gas limit for state transactions
	StateTransactionGasLimit = 1000000
)

// TxType is the transaction type.
type TxType byte

// List of supported transaction types
const (
	LegacyTx     TxType = 0x0
	StateTx      TxType = 0x7f
	DynamicFeeTx TxType = 0x8f
)

// Transaction types.
const (
	LegacyTxType = iota
	AccessListTxType
	DynamicFeeTxType
)

func txTypeFromByte(b byte) (TxType, error) {
	tt := TxType(b)

	switch tt {
	// case LegacyTx, StateTx, DynamicFeeTx:
	case AccessListTxType, StateTx: //Todo: big problem, temp compatible
		return StateTx, nil
	case DynamicFeeTxType, DynamicFeeTx:
		return DynamicFeeTx, nil
	default:
		return tt, fmt.Errorf("unknown transaction type: %d", b)
	}
}

// Config are the configuration options for structured logger the EVM
type LoggerConfig struct {
	EnableMemory     bool // enable memory capture
	DisableStack     bool // disable stack capture
	DisableStorage   bool // disable storage capture
	EnableReturnData bool // enable return data capture
	Debug            bool // print output during capture end
	Limit            int  // maximum length of output, but zero means unlimited
}

// TxData is the underlying data of a transaction.
//
// This is implemented by DynamicFeeTx, LegacyTx and AccessListTx.
type TxData interface {
	txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)
}

type Transaction struct {
	ChainId   *big.Int
	Nonce     uint64
	GasPrice  *big.Int
	GasTipCap *big.Int
	GasFeeCap *big.Int
	Gas       uint64
	To        *Address
	Value     *big.Int
	Input     []byte
	V         *big.Int
	R         *big.Int
	S         *big.Int
	Hash      Hash
	From      Address

	Type TxType

	// Cache
	size atomic.Value

	LoggerConfig *LoggerConfig
}

func (t *Transaction) GetGasPrice() *big.Int {
	if t.GasPrice != nil {
		return t.GasPrice
	}

	return t.GasFeeCap
}

func (t *Transaction) IsContractCreation() bool {
	return t.To == nil
}

// ComputeHash computes the hash of the transaction
func (t *Transaction) ComputeHash() *Transaction {
	ar := marshalArenaPool.Get()
	hash := keccak.DefaultKeccakPool.Get()

	if t.GasPrice == nil {
		if t.Type == DynamicFeeTx {
			hash.Write([]byte{2})
		} else {
			hash.Write([]byte{1})
		}
	}

	v := t.MarshalRLPWith(ar)
	hash.WriteRlp(t.Hash[:0], v)

	marshalArenaPool.Put(ar)
	keccak.DefaultKeccakPool.Put(hash)

	return t
}

func (t *Transaction) Copy() *Transaction {
	tt := new(Transaction)
	*tt = *t

	tt.GasPrice = new(big.Int)
	if t.GasPrice != nil {
		tt.GasPrice.Set(t.GasPrice)
	}

	tt.Value = new(big.Int)
	if t.Value != nil {
		tt.Value.Set(t.Value)
	}

	if t.R != nil {
		tt.R = new(big.Int)
		tt.R = big.NewInt(0).SetBits(t.R.Bits())
	}

	if t.S != nil {
		tt.S = new(big.Int)
		tt.S = big.NewInt(0).SetBits(t.S.Bits())
	}

	tt.Input = make([]byte, len(t.Input))
	copy(tt.Input[:], t.Input[:])

	return tt
}

// Cost returns gas * gasPrice + value
func (t *Transaction) Cost() *big.Int { //Todo adpter eip1559
	// total := new(big.Int).Mul(t.GasPrice, new(big.Int).SetUint64(t.Gas))
	total := new(big.Int).Mul(t.GetGasPrice(), new(big.Int).SetUint64(t.Gas)) //Todo: Important, need improve
	total.Add(total, t.Value)

	return total
}

func (t *Transaction) Size() uint64 {
	if size := t.size.Load(); size != nil {
		sizeVal, ok := size.(uint64)
		if !ok {
			return 0
		}

		return sizeVal
	}

	size := uint64(len(t.MarshalRLP()))
	t.size.Store(size)

	return size
}

//Todo: Is this should need ?
func (t *Transaction) ExceedsBlockGasLimit(blockGasLimit uint64) bool {
	return t.Gas > blockGasLimit
}

func (t *Transaction) IsUnderpriced(priceLimit uint64) bool { //Todo: Important, need improve
	if t.GetGasPrice() != nil {
		return t.GetGasPrice().Cmp(big.NewInt(0).SetUint64(priceLimit)) < 0
	}

	return t.GetGasPrice().Cmp(big.NewInt(0).SetUint64(priceLimit)) < 0
}

func (t *Transaction) SetLoggerConfig(config *LoggerConfig) {
	t.LoggerConfig = config
}

// copyAddressPtr copies an address.
func copyAddressPtr(a *Address) *Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}
