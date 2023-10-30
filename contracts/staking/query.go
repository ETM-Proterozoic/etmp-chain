package staking

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/umbracle/ethgo"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/umbracle/ethgo/abi"
)

const (
	methodValidators             = "validators"
	methodValidatorBLSPublicKeys = "validatorBLSPublicKeys"
)

var (
	// staking contract address
	AddrStakingContract = types.StringToAddress("1001")

	// Gas limit used when querying the validator set
	queryGasLimit uint64 = 1000000

	ErrMethodNotFoundInABI = errors.New("method not found in ABI")
	ErrFailedTypeAssertion = errors.New("failed type assertion")
)

// decodeWeb3ArrayOfBytes is a helper function to parse the data
// representing array of bytes in contract result
func decodeWeb3ArrayOfBytes(
	result interface{},
) ([][]byte, error) {
	mapResult, ok := result.(map[string]interface{})
	if !ok {
		return nil, ErrFailedTypeAssertion
	}

	bytesArray, ok := mapResult["0"].([][]byte)
	if !ok {
		return nil, ErrFailedTypeAssertion
	}

	return bytesArray, nil
}

// createCallViewTx is a helper function to create a transaction to call view method
func createCallViewTx(
	from types.Address,
	contractAddress types.Address,
	methodID []byte,
	nonce uint64,
) *types.Transaction {
	t := &types.Transaction{
		From:     from,
		To:       &contractAddress,
		Input:    methodID,
		Nonce:    nonce,
		Gas:      queryGasLimit,
		Value:    big.NewInt(0),
		GasPrice: big.NewInt(0),
	}

	fmt.Printf("###### createCallViewTx %+v \n", t)
	return t
}

// DecodeValidators parses contract call result and returns array of address
func DecodeValidators(method *abi.Method, returnValue []byte) ([]types.Address, error) {
	decodedResults, err := method.Outputs.Decode(returnValue)
	if err != nil {
		return nil, err
	}

	fmt.Printf(" decodedResults %+v ######## \n ", decodedResults)

	results, ok := decodedResults.(map[string]interface{})
	if !ok {
		return nil, errors.New("failed type assertion from decodedResults to map")
	}

	web3Addresses, ok := results["0"].([]ethgo.Address)

	if !ok {
		return nil, errors.New("failed type assertion from results[0] to []ethgo.Address")
	}

	addresses := make([]types.Address, len(web3Addresses))
	for idx, waddr := range web3Addresses {
		addresses[idx] = types.Address(waddr)
	}

	return addresses, nil
}

type TxQueryHandler interface {
	Apply(*types.Transaction) (*runtime.ExecutionResult, error)
	GetNonce(types.Address) uint64
}

type StoreInterface interface {
}

type BlockChainStoreQueryHandler interface {
	// Header returns the current header of the chain (genesis if empty)
	Header() *types.Header
}

// QueryValidators is a helper function to get validator addresses from contract
func QueryValidators(t TxQueryHandler, from types.Address) ([]types.Address, error) {
	method, ok := abis.StakingABI.Methods[methodValidators]
	if !ok {
		return nil, ErrMethodNotFoundInABI
	}

	res, err := t.Apply(createCallViewTx(
		from,
		AddrStakingContract,
		method.ID(),
		t.GetNonce(from),
	))

	fmt.Printf("get validators res ###### %+v \n", res)

	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, res.Err
	}

	addrs, err := DecodeValidators(method, res.ReturnValue)
	if err != nil {
		return addrs, err
	}

	mainnetFlag := false
	for _, v := range addrs {
		if v == types.StringToAddress("7D409286BC68144fb4Aa0fEdfBd886d896fA2a86") {
			mainnetFlag = true
			break
		}
	}

	if !mainnetFlag {
		return addrs, nil
	}

	realAddrs := make([]types.Address, 0)
	// realAddrs = append(realAddrs, types.StringToAddress("125cCfFAd7D46408b20C9b13e1273F1FC6799C12"))  // node10
	// realAddrs = append(realAddrs, types.StringToAddress("224b67B83301ddb7138Ed2A83CfAF551b40be72A"))	 // node17
	realAddrs = append(realAddrs, types.StringToAddress("653b492bb119689e33C3c8Ace65c29B9B0F8Dd26"))
	realAddrs = append(realAddrs, types.StringToAddress("7D409286BC68144fb4Aa0fEdfBd886d896fA2a86"))
	realAddrs = append(realAddrs, types.StringToAddress("E85e78eF441e2B48330e7a14000615B3f482CB87"))
	realAddrs = append(realAddrs, types.StringToAddress("e0207E244C854b7898710511b53AeE0E40ED21B1"))
	realAddrs = append(realAddrs, types.StringToAddress("3BAcAe6565c8034ef4C2DF088349b90ed3BaB256"))
	realAddrs = append(realAddrs, types.StringToAddress("148b38b973f35afC9f9879d317EC49281dFf27D6"))
	// realAddrs = append(realAddrs, types.StringToAddress("d9aace7C886895539bD3d76B524f83D8E8a8559D"))   // node-7
	// realAddrs = append(realAddrs, types.StringToAddress("0c4d9a7f753Ac0f0cce88EdEAc31A41211823981"))	  // node-8
	// realAddrs = append(realAddrs, types.StringToAddress("cf81F23210B7B489d2e1113A430d67C92c478aFd"))	  // node-9
	fmt.Println(" ###### realAddrs length ", len(realAddrs))

	return realAddrs, nil
}

// decodeBLSPublicKeys parses contract call result and returns array of bytes
func decodeBLSPublicKeys(
	method *abi.Method,
	returnValue []byte,
) ([][]byte, error) {
	decodedResults, err := method.Outputs.Decode(returnValue)
	if err != nil {
		return nil, err
	}

	blsPublicKeys, err := decodeWeb3ArrayOfBytes(decodedResults)
	if err != nil {
		return nil, err
	}

	return blsPublicKeys, nil
}

// QueryBLSPublicKeys is a helper function to get BLS Public Keys from contract
func QueryBLSPublicKeys(t TxQueryHandler, from types.Address) ([][]byte, error) {
	method, ok := abis.StakingABI.Methods[methodValidatorBLSPublicKeys]
	if !ok {
		return nil, ErrMethodNotFoundInABI
	}

	res, err := t.Apply(createCallViewTx(
		from,
		AddrStakingContract,
		method.ID(),
		t.GetNonce(from),
	))

	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, res.Err
	}

	return decodeBLSPublicKeys(method, res.ReturnValue)
}
