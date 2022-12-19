package helper

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/secrets"
	"github.com/0xPolygon/polygon-edge/secrets/awskms"
	"github.com/0xPolygon/polygon-edge/secrets/awsssm"
	"github.com/0xPolygon/polygon-edge/secrets/gcpssm"
	"github.com/0xPolygon/polygon-edge/secrets/hashicorpvault"
	"github.com/0xPolygon/polygon-edge/secrets/local"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// SetupLocalSecretsManager is a helper method for boilerplate local secrets manager setup
func SetupLocalSecretsManager(dataDir string) (secrets.SecretsManager, error) {
	return local.SecretsManagerFactory(
		nil, // Local secrets manager doesn't require a config
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
			Extra: map[string]interface{}{
				secrets.Path: dataDir,
			},
		},
	)
}

// SetupHashicorpVault is a helper method for boilerplate hashicorp vault secrets manager setup
func SetupHashicorpVault(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return hashicorpvault.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// SetupAWSSSM is a helper method for boilerplate aws ssm secrets manager setup
func SetupAWSSSM(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return awsssm.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// SetupGCPSSM is a helper method for boilerplate Google Cloud Computing secrets manager setup
func SetupGCPSSM(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return gcpssm.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// SetupAwsKms is a helper method for boilerplate Google Cloud Computing secrets manager setup
func SetupAwsKms(
	secretsConfig *secrets.SecretsManagerConfig, dataDir string,
) (secrets.SecretsManager, error) {
	subDirectories := []string{secrets.ConsensusFolderLocal, secrets.NetworkFolderLocal}

	// Check if the sub-directories exist / are already populated
	for _, subDirectory := range subDirectories {
		if common.DirectoryExists(filepath.Join(dataDir, subDirectory)) {
			return nil,
				fmt.Errorf(
					"directory %s has previously initialized secrets data",
					dataDir,
				)
		}
	}

	return awskms.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
			Extra: map[string]interface{}{
				secrets.Path: dataDir,
			},
		},
	)
}

func InitValidatorKey(secretsManager secrets.SecretsManager) (*ecdsa.PrivateKey, error) {
	// Generate the IBFT validator private key
	validatorKey, validatorKeyEncoded, keyErr := crypto.GenerateAndEncodePrivateKey()
	if keyErr != nil {
		return nil, keyErr
	}

	address := crypto.PubKeyToAddress(&validatorKey.PublicKey)

	// Write the validator private key to the secrets manager storage
	if setErr := secretsManager.SetSecret(
		secrets.ValidatorKey,
		validatorKeyEncoded,
	); setErr != nil {
		return types.ZeroAddress, setErr
	}

	return address, nil
}

// InitECDSAValidatorKey creates new ECDSA key and set as a validator key
func InitECDSAValidatorKey(secretsManager secrets.SecretsManager) (types.Address, error) {
	if secretsManager.HasSecret(secrets.ValidatorKey) {
		return types.ZeroAddress, fmt.Errorf(`secrets "%s" has been already initialized`, secrets.ValidatorKey)
	}

	validatorKey, validatorKeyEncoded, err := crypto.GenerateAndEncodeECDSAPrivateKey()
	if err != nil {
		return types.ZeroAddress, err
	}

	address := crypto.PubKeyToAddress(&validatorKey.PublicKey)

	// Write the validator private key to the secrets manager storage
	if setErr := secretsManager.SetSecret(
		secrets.ValidatorKey,
		validatorKeyEncoded,
	); setErr != nil {
		return types.ZeroAddress, setErr
	}

	return address, nil
}

func InitBLSValidatorKey(secretsManager secrets.SecretsManager) ([]byte, error) {
	if secretsManager.HasSecret(secrets.ValidatorBLSKey) {
		return nil, fmt.Errorf(`secrets "%s" has been already initialized`, secrets.ValidatorBLSKey)
	}

	blsSecretKey, blsSecretKeyEncoded, err := crypto.GenerateAndEncodeBLSSecretKey()
	if err != nil {
		return nil, err
	}

	// Write the validator private key to the secrets manager storage
	if setErr := secretsManager.SetSecret(
		secrets.ValidatorBLSKey,
		blsSecretKeyEncoded,
	); setErr != nil {
		return nil, setErr
	}

	pubkeyBytes, err := crypto.BLSSecretKeyToPubkeyBytes(blsSecretKey)
	if err != nil {
		return nil, err
	}

	return pubkeyBytes, nil
}

func InitNetworkingPrivateKey(secretsManager secrets.SecretsManager) (libp2pCrypto.PrivKey, error) {
	if secretsManager.HasSecret(secrets.NetworkKey) {
		return nil, fmt.Errorf(`secrets "%s" has been already initialized`, secrets.NetworkKey)
	}

	// Generate the libp2p private key
	libp2pKey, libp2pKeyEncoded, keyErr := network.GenerateAndEncodeLibp2pKey()
	if keyErr != nil {
		return nil, keyErr
	}

	// Write the networking private key to the secrets manager storage
	if setErr := secretsManager.SetSecret(
		secrets.NetworkKey,
		libp2pKeyEncoded,
	); setErr != nil {
		return nil, setErr
	}

	return libp2pKey, keyErr
}
