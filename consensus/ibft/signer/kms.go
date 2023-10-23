package signer

import (
	"github.com/0xPolygon/polygon-edge/secrets"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/0xPolygon/polygon-edge/validators"
)

// KmsKeyManager is a module that holds ECDSA key
// and implements methods of signing by this key
type KmsKeyManager struct {
	manager secrets.SecretsManager
	address types.Address
	chainId int
}

// NewKmsKeyManager initializes KmsKeyManager by the ECDSA key loaded from SecretsManager
func NewKmsKeyManager(manager secrets.SecretsManager, chainId int) (KeyManager, error) {
	k := &KmsKeyManager{
		manager: manager,
		chainId: chainId,
	}

	info, err := manager.GetSecretInfo(secrets.ValidatorKey)
	if err != nil {
		return nil, err
	}
	k.address = types.StringToAddress(info.Address)

	return k, nil
}

// Type returns the validator type KeyManager supports
func (k *KmsKeyManager) Type() validators.ValidatorType {
	return validators.ECDSAValidatorType
}

// Address returns the address of KeyManager
func (k *KmsKeyManager) Address() types.Address {
	return k.address
}

// NewEmptyValidators returns empty validator collection KmsKeyManager uses
func (k *KmsKeyManager) NewEmptyValidators() validators.Validators {
	return validators.NewECDSAValidatorSet()
}

// NewEmptyCommittedSeals returns empty CommittedSeals KmsKeyManager uses
func (k *KmsKeyManager) NewEmptyCommittedSeals() Seals {
	return &SerializedSeal{}
}

// SignProposerSeal signs the given message by ECDSA key the KmsKeyManager holds for ProposerSeal
func (k *KmsKeyManager) SignProposerSeal(message []byte) ([]byte, error) {
	return k.manager.SignBySecret(secrets.ValidatorKey, k.chainId, message)

}

// SignProposerSeal signs the given message by ECDSA key the KmsKeyManager holds for committed seal
func (k *KmsKeyManager) SignCommittedSeal(message []byte) ([]byte, error) {
	return k.manager.SignBySecret(secrets.ValidatorKey, k.chainId, message)
}

// VerifyCommittedSeal verifies a committed seal
func (k *KmsKeyManager) VerifyCommittedSeal(
	vals validators.Validators,
	address types.Address,
	signature []byte,
	message []byte,
) error {
	if vals.Type() != k.Type() {
		return ErrInvalidValidators
	}

	signer, err := k.Ecrecover(signature, message)
	if err != nil {
		return ErrInvalidSignature
	}

	if address != signer {
		return ErrSignerMismatch
	}

	// if !vals.Includes(address) {
	// 	return ErrNonValidatorCommittedSeal
	// }

	return nil
}

func (k *KmsKeyManager) GenerateCommittedSeals(
	sealMap map[types.Address][]byte,
	_ validators.Validators,
) (Seals, error) {
	seals := [][]byte{}

	for _, seal := range sealMap {
		if len(seal) != IstanbulExtraSeal {
			return nil, ErrInvalidCommittedSealLength
		}

		seals = append(seals, seal)
	}

	serializedSeal := SerializedSeal(seals)

	return &serializedSeal, nil
}

func (k *KmsKeyManager) VerifyCommittedSeals(
	rawCommittedSeal Seals,
	digest []byte,
	vals validators.Validators,
) (int, error) {
	committedSeal, ok := rawCommittedSeal.(*SerializedSeal)
	if !ok {
		return 0, ErrInvalidCommittedSealType
	}

	if vals.Type() != k.Type() {
		return 0, ErrInvalidValidators
	}

	return k.verifyCommittedSealsImpl(committedSeal, digest, vals)
}

func (k *KmsKeyManager) SignIBFTMessage(msg []byte) ([]byte, error) {
	return k.manager.SignBySecret(secrets.ValidatorKey, k.chainId, msg)
}

func (k *KmsKeyManager) Ecrecover(sig, digest []byte) (types.Address, error) {
	return ecrecover(sig, digest)
}

func (k *KmsKeyManager) verifyCommittedSealsImpl(
	committedSeal *SerializedSeal,
	msg []byte,
	validators validators.Validators,
) (int, error) {
	numSeals := committedSeal.Num()
	// debug.PrintStack()
	if numSeals == 0 {
		return 0, ErrEmptyCommittedSeals
	}

	visited := make(map[types.Address]bool)

	for _, seal := range *committedSeal {
		addr, err := k.Ecrecover(seal, msg)
		if err != nil {
			return 0, err
		}

		if visited[addr] {
			return 0, ErrRepeatedCommittedSeal
		}

		if !validators.Includes(addr) {
			return 0, ErrNonValidatorCommittedSeal
		}

		visited[addr] = true
	}

	return numSeals, nil
}
